package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/axiom-idp/axiom/internal/config"
	"github.com/sirupsen/logrus"
)

type aiBackend interface {
	Query(ctx context.Context, request aiQueryRequest) (string, string, error)
}

type aiQueryRequest struct {
	Query     string
	Services  []demoService
	Focus     *catalogServiceView
	Portfolio portfolioIntelligence
	Intent    string
	UserID    string
	Roles     []string
}

type localAIBackend struct{}

type openAICompatibleBackend struct {
	backend   string
	baseURL   string
	apiKey    string
	model     string
	maxTokens int
	client    *http.Client
}

func newAIBackend(cfg *config.Config, logger *logrus.Logger) aiBackend {
	if cfg == nil {
		return localAIBackend{}
	}

	if strings.EqualFold(cfg.AIBackend, "ollama") || strings.EqualFold(cfg.AIBackend, "openai") {
		return &openAICompatibleBackend{
			backend:   strings.ToLower(strings.TrimSpace(cfg.AIBackend)),
			baseURL:   strings.TrimRight(cfg.AIBaseURL, "/"),
			apiKey:    strings.TrimSpace(cfg.AIAPIKey),
			model:     cfg.AIModel,
			maxTokens: cfg.AIMaxTokens,
			client: &http.Client{
				Timeout: cfg.AITimeout,
			},
		}
	}

	return localAIBackend{}
}

func (localAIBackend) Query(ctx context.Context, request aiQueryRequest) (string, string, error) {
	queryLower := strings.ToLower(strings.TrimSpace(request.Query))
	if queryLower == "" {
		return "Please provide a question or request.", "local", nil
	}

	var matches []string
	for _, service := range request.Services {
		name := strings.ToLower(service.Name)
		desc := strings.ToLower(service.Description)
		if strings.Contains(name, queryLower) || strings.Contains(queryLower, name) || strings.Contains(desc, queryLower) {
			matches = append(matches, service.Name)
		}
	}

	switch {
	case strings.Contains(queryLower, "brief"), strings.Contains(queryLower, "decision pack"), strings.Contains(queryLower, "operator brief"):
		if request.Focus != nil {
			return fmt.Sprintf(
				"Release brief for %s: %s. Next best action: %s. Evidence pack contains %d items. Portfolio context: %d ready, %d blocked, %d owner gaps.",
				request.Focus.Service.Name,
				request.Focus.Intelligence.ReleaseReadiness.Reason,
				renderActionList(request.Focus.Intelligence.NextSteps, 1),
				len(request.Focus.Intelligence.EvidencePack),
				request.Portfolio.ReadyCount,
				request.Portfolio.BlockedCount,
				request.Portfolio.OwnerGapCount,
			), "local", nil
		}
		return fmt.Sprintf(
			"Generate a release brief from the catalog: %d services are indexed, %d are ready, %d need attention, and %d are blocked.",
			request.Portfolio.TotalServices,
			request.Portfolio.ReadyCount,
			request.Portfolio.WatchCount,
			request.Portfolio.BlockedCount,
		), "local", nil
	case strings.Contains(queryLower, "bsi c5"), strings.Contains(queryLower, "compliance"), strings.Contains(queryLower, "evidence"):
		if request.Focus != nil {
			return fmt.Sprintf("%s is %s with %d required evidence items. Review the evidence pack, confirm ownership, and attach the audit artifacts before approval.", request.Focus.Service.Name, request.Focus.Intelligence.ReleaseReadiness.State, len(request.Focus.Intelligence.EvidencePack)), "local", nil
		}
		return fmt.Sprintf("Use the catalog analysis to review evidence packs, ownership drift, and release readiness before approval. %d services are indexed.", len(request.Services)), "local", nil
	case strings.Contains(queryLower, "risk"), strings.Contains(queryLower, "release"), strings.Contains(queryLower, "deploy"):
		if request.Focus != nil {
			return fmt.Sprintf("%s is %s with a risk score of %d. Next steps: %s.", request.Focus.Service.Name, request.Focus.Intelligence.ReleaseReadiness.State, request.Focus.Intelligence.ReleaseReadiness.Score, renderActionList(request.Focus.Intelligence.NextSteps, 2)), "local", nil
		}
		return fmt.Sprintf("Current release guidance: %d services are indexed, %d are ready, %d need attention, and %d are blocked.", request.Portfolio.TotalServices, request.Portfolio.ReadyCount, request.Portfolio.WatchCount, request.Portfolio.BlockedCount), "local", nil
	case strings.Contains(queryLower, "owner"), strings.Contains(queryLower, "ownership"), strings.Contains(queryLower, "drift"):
		if request.Focus != nil {
			return fmt.Sprintf("%s ownership is %s. %s", request.Focus.Service.Name, request.Focus.Intelligence.OwnershipDrift.State, renderActionList(request.Focus.Intelligence.NextSteps, 2)), "local", nil
		}
		return "Review catalog ownership metadata and align CODEOWNERS, escalation paths, and service records before approval.", "local", nil
	case len(matches) > 0:
		return fmt.Sprintf("Matching services: %s. Use the catalog to inspect readiness and ask for a service-specific risk or evidence summary.", strings.Join(matches, ", ")), "local", nil
	default:
		return fmt.Sprintf("I can help with release risk, service ownership, audit evidence, and rollout guidance. Your request was processed for user %s with roles %s.", fallbackUser(request.UserID), strings.Join(defaultIfEmpty(request.Roles, []string{"viewer"}), ", ")), "local", nil
	}
}

func (o *openAICompatibleBackend) Query(ctx context.Context, request aiQueryRequest) (string, string, error) {
	if strings.TrimSpace(request.Query) == "" {
		return "", o.backend, nil
	}

	prompt := buildOpenAICompatiblePrompt(request)
	body := map[string]interface{}{
		"model": o.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are Axiom IDP's production operations assistant. Answer concisely, ground responses in provided structured analysis, and prioritize release readiness, ownership, compliance evidence, observability, and operational next steps.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.2,
		"max_tokens":  o.maxTokens,
		"stream":      false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", o.backend, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAICompatibleChatCompletionsURL(o.baseURL), bytes.NewReader(payload))
	if err != nil {
		return "", o.backend, err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return "", o.backend, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", o.backend, fmt.Errorf("%s-compatible endpoint returned status %d", o.backend, resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", o.backend, err
	}

	answer := ""
	if len(result.Choices) > 0 {
		answer = strings.TrimSpace(result.Choices[0].Message.Content)
	}
	if answer == "" {
		return "", o.backend, fmt.Errorf("%s-compatible endpoint returned an empty response", o.backend)
	}

	return answer, o.backend, nil
}

func buildOpenAICompatiblePrompt(request aiQueryRequest) string {
	var serviceLines []string
	for _, service := range request.Services {
		serviceLines = append(serviceLines, fmt.Sprintf("- %s | owner=%s | team=%s | status=%s | tier=%s | dependencies=%s", service.Name, service.Owner, service.Team, service.Status, service.Tier, strings.Join(service.Dependencies, ", ")))
	}

	report := map[string]interface{}{
		"intent":    request.Intent,
		"user_id":   fallbackUser(request.UserID),
		"roles":     defaultIfEmpty(request.Roles, []string{"viewer"}),
		"focus":     request.Focus,
		"portfolio": request.Portfolio,
		"services":  request.Services,
	}

	reportJSON, _ := json.Marshal(report)

	return fmt.Sprintf(`You are Axiom IDP's production operations assistant.

User:
- id: %s
- roles: %s

Structured analysis:
%s

Known services:
%s

Instructions:
- Answer concisely.
- Focus on deployment readiness, release risk, ownership, security posture, and compliance evidence.
- If the question asks for a release brief, provide the decision, missing evidence, and the next best action in one concise answer.
- If the question asks for an action, give concrete next steps.
- Use the structured analysis as the source of truth.

User question:
%s`, fallbackUser(request.UserID), strings.Join(defaultIfEmpty(request.Roles, []string{"viewer"}), ", "), string(reportJSON), strings.Join(serviceLines, "\n"), strings.TrimSpace(request.Query))
}

func openAICompatibleChatCompletionsURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/chat/completions"
	}
	return baseURL + "/v1/chat/completions"
}

func fallbackUser(userID string) string {
	if strings.TrimSpace(userID) == "" {
		return "anonymous"
	}
	return userID
}

func defaultIfEmpty(values []string, fallback []string) []string {
	if len(values) == 0 {
		return fallback
	}
	return values
}

func queryAI(ctx context.Context, backend aiBackend, request aiQueryRequest, logger *logrus.Logger) (string, string) {
	if backend == nil {
		backend = localAIBackend{}
	}

	answer, source, err := backend.Query(ctx, request)
	if err != nil {
		if logger != nil {
			logger.WithError(err).Warn("ai backend query failed, falling back to local response")
		}
		localAnswer, _, _ := localAIBackend{}.Query(ctx, request)
		return localAnswer, "local-fallback"
	}

	if strings.TrimSpace(answer) == "" {
		if logger != nil {
			logger.Warn("ai backend returned an empty response, falling back to local response")
		}
		localAnswer, _, _ := localAIBackend{}.Query(ctx, request)
		return localAnswer, "local-fallback"
	}

	return answer, source
}

func newAIRequestContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return context.WithTimeout(ctx, timeout)
}
