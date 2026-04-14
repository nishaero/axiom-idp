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
	Query(ctx context.Context, query string, services []demoService, userID string, roles []string) (string, string, error)
}

type localAIBackend struct{}

type ollamaAIBackend struct {
	baseURL   string
	model     string
	maxTokens int
	client    *http.Client
	logger    *logrus.Logger
}

func newAIBackend(cfg *config.Config, logger *logrus.Logger) aiBackend {
	if cfg == nil {
		return localAIBackend{}
	}

	if strings.EqualFold(cfg.AIBackend, "ollama") {
		return &ollamaAIBackend{
			baseURL:   strings.TrimRight(cfg.AIBaseURL, "/"),
			model:     cfg.AIModel,
			maxTokens: cfg.AIMaxTokens,
			client: &http.Client{
				Timeout: cfg.AITimeout,
			},
			logger: logger,
		}
	}

	return localAIBackend{}
}

func (localAIBackend) Query(ctx context.Context, query string, services []demoService, userID string, roles []string) (string, string, error) {
	queryLower := strings.ToLower(strings.TrimSpace(query))
	if queryLower == "" {
		return "Please provide a question or request.", "local", nil
	}

	var matches []string
	for _, service := range services {
		name := strings.ToLower(service.Name)
		desc := strings.ToLower(service.Description)
		if strings.Contains(name, queryLower) || strings.Contains(queryLower, name) || strings.Contains(desc, queryLower) {
			matches = append(matches, service.Name)
		}
	}

	switch {
	case strings.Contains(queryLower, "bsi c5"), strings.Contains(queryLower, "compliance"), strings.Contains(queryLower, "evidence"):
		return "Use the settings and dashboard views to review release evidence, ownership, and runtime posture before approval. For BSI C5 readiness, confirm owner assignment, security headers, authenticated access, and deployment validation are green.", "local", nil
	case strings.Contains(queryLower, "risk"), strings.Contains(queryLower, "release"), strings.Contains(queryLower, "deploy"):
		return fmt.Sprintf("Current release guidance: %d services are indexed, and the highest-risk item is any degraded service or owner gap. Review identity-gateway first for access controls, then confirm payments-api and orders-worker readiness before rollout.", len(services)), "local", nil
	case len(matches) > 0:
		return fmt.Sprintf("Matching services: %s. Use the catalog to inspect readiness and ask for a service-specific risk or evidence summary.", strings.Join(matches, ", ")), "local", nil
	default:
		return fmt.Sprintf("I can help with release risk, service ownership, audit evidence, and rollout guidance. Your request was processed for user %s with roles %s.", fallbackUser(userID), strings.Join(defaultIfEmpty(roles, []string{"viewer"}), ", ")), "local", nil
	}
}

func (o *ollamaAIBackend) Query(ctx context.Context, query string, services []demoService, userID string, roles []string) (string, string, error) {
	if strings.TrimSpace(query) == "" {
		return "", "ollama", nil
	}

	prompt := buildOllamaPrompt(query, services, userID, roles)
	body := map[string]interface{}{
		"model":  o.model,
		"prompt": prompt,
		"stream": false,
		"think":  false,
		"options": map[string]interface{}{
			"temperature": 0.2,
			"num_predict": o.maxTokens,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", "ollama", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return "", "ollama", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", "ollama", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "ollama", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
		Thinking string `json:"thinking"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "ollama", err
	}

	answer := strings.TrimSpace(result.Response)
	if answer == "" {
		if strings.TrimSpace(result.Thinking) != "" {
			return "", "ollama", fmt.Errorf("ollama returned only thinking output")
		}
		return "", "ollama", fmt.Errorf("ollama returned an empty response")
	}

	return answer, "ollama", nil
}

func buildOllamaPrompt(query string, services []demoService, userID string, roles []string) string {
	var serviceLines []string
	for _, service := range services {
		serviceLines = append(serviceLines, fmt.Sprintf("- %s | owner=%s | status=%s | description=%s", service.Name, service.Owner, service.Status, service.Description))
	}

	return fmt.Sprintf(`You are Axiom IDP's production operations assistant.

User:
- id: %s
- roles: %s

Known services:
%s

Instructions:
- Answer concisely.
- Focus on deployment readiness, release risk, ownership, security posture, and compliance evidence.
- If the question asks for an action, give concrete next steps.
- If information is missing, say what must be checked next.

User question:
%s`, fallbackUser(userID), strings.Join(defaultIfEmpty(roles, []string{"viewer"}), ", "), strings.Join(serviceLines, "\n"), strings.TrimSpace(query))
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

func queryAI(ctx context.Context, backend aiBackend, query string, services []demoService, userID string, roles []string, logger *logrus.Logger) (string, string) {
	if backend == nil {
		backend = localAIBackend{}
	}

	answer, source, err := backend.Query(ctx, query, services, userID, roles)
	if err != nil {
		if logger != nil {
			logger.WithError(err).Warn("ai backend query failed, falling back to local response")
		}
		localAnswer, _, _ := localAIBackend{}.Query(ctx, query, services, userID, roles)
		return localAnswer, "local-fallback"
	}

	if strings.TrimSpace(answer) == "" {
		if logger != nil {
			logger.Warn("ai backend returned an empty response, falling back to local response")
		}
		localAnswer, _, _ := localAIBackend{}.Query(ctx, query, services, userID, roles)
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
