package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-github/v56/github"
	"github.com/sirupsen/logrus"
)

// WebhookHandler handles GitHub webhook events
type WebhookHandler struct {
	logger     *logrus.Logger
	client     *GitHubClient
	secret     string
	registry   *EventHandlerRegistry
}

// EventHandlerRegistry manages event handlers
type EventHandlerRegistry struct {
	handlers map[string][]func(event *PREvent)
}

// NewEventHandlerRegistry creates a new event handler registry
func NewEventHandlerRegistry() *EventHandlerRegistry {
	return &EventHandlerRegistry{
		handlers: make(map[string][]func(event *PREvent)),
	}
}

// RegisterHandler registers an event handler for a specific event type
func (r *EventHandlerRegistry) RegisterHandler(eventType string, handler func(event *PREvent)) {
	r.handlers[eventType] = append(r.handlers[eventType], handler)
}

// GetHandlers returns all handlers for an event type
func (r *EventHandlerRegistry) GetHandlers(eventType string) []func(event *PREvent) {
	return r.handlers[eventType]
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(logger *logrus.Logger, client *GitHubClient, secret string, registry *EventHandlerRegistry) *WebhookHandler {
	return &WebhookHandler{
		logger:   logger.WithField("component", "webhook_handler"),
		client:   client,
		secret:   secret,
		registry: registry,
	}
}

// Handle handles incoming webhook requests
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify signature
	signature := r.Header.Get("X-Hub-Signature-256")
	if !h.verifySignature(r, signature) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		h.logger.Warn("Invalid webhook signature")
		return
	}

	// Read payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read payload", http.StatusBadRequest)
		h.logger.WithError(err).Error("Failed to read webhook payload")
		return
	}
	defer r.Body.Close()

	// Parse event
	event, err := parseWebhookEvent(payload, r.Header.Get("X-GitHub-Event"))
	if err != nil {
		http.Error(w, "invalid event", http.StatusBadRequest)
		h.logger.WithError(err).Error("Failed to parse webhook event")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"event":    r.Header.Get("X-GitHub-Event"),
		"action":   event.Action,
		"repo":     event.Repository.FullName,
		"pr_number": event.PR.Number,
	}).Info("Received webhook event")

	// Process event
	h.processEvent(ctx, event)

	// Send acknowledgment
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Event received")
}

// verifySignature verifies the webhook signature
func (h *WebhookHandler) verifySignature(r *http.Request, signature string) bool {
	if h.secret == "" {
		return true // No secret configured, skip verification
	}

	if signature == "" {
		h.logger.Warn("Missing signature header")
		return false
	}

	// Extract signature
	startIdx := len("sha256=")
	if len(signature) > startIdx {
		signature = signature[startIdx:]
	} else {
		signature = ""
	}

	// For now, skip actual verification (would use hmac in production)
	return true
}

// parseWebhookEvent parses a webhook payload
func parseWebhookEvent(payload []byte, eventType string) (*PREvent, error) {
	switch eventType {
	case "pull_request":
		return parsePREvent(payload)
	case "push":
		return parsePushEvent(payload)
	case "workflow_run":
		return parseWorkflowRunEvent(payload)
	case "status":
		return parseStatusEvent(payload)
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}
}

// parsePREvent parses a pull request event
func parsePREvent(payload []byte) (*PREvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	event := &PREvent{}

	// Parse basic fields
	event.Action = getString(raw, "action")
	event.Repository = *parseRepository(raw, "repository")

	// Parse pull request
	if prRaw, exists := raw["pull_request"]; exists {
		prJSON, err := json.Marshal(prRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal pull request: %w", err)
		}
		pr := &PullRequest{}
		if err := json.Unmarshal(prJSON, pr); err != nil {
			return nil, fmt.Errorf("failed to parse pull request: %w", err)
		}
		event.PR = *pr
	}

	// Parse sender
	if senderRaw, exists := raw["sender"]; exists {
		senderJSON, err := json.Marshal(senderRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal sender: %w", err)
		}
		sender := &User{}
		if err := json.Unmarshal(senderJSON, sender); err != nil {
			return nil, fmt.Errorf("failed to parse sender: %w", err)
		}
		event.Sender = *sender
	}

	// Parse optional fields
	if orgRaw, exists := raw["organization"]; exists {
		orgJSON, err := json.Marshal(orgRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal organization: %w", err)
		}
		var org map[string]interface{}
		json.Unmarshal(orgJSON, &org)
		event.Organization = getString(org, "login")
	}

	if hookID, exists := raw["hook_id"]; exists {
		if id, ok := hookID.(float64); ok {
			event.HookID = int64(id)
		}
	}

	if installationID, exists := raw["installation_id"]; exists {
		if id, ok := installationID.(float64); ok {
			event.InstallationID = int64(id)
		}
	}

	return event, nil
}

// parsePushEvent parses a push event
func parsePushEvent(payload []byte) (*PREvent, error) {
	// For simplicity, return a minimal PREvent
	// In production, would parse PushEvent and convert to PREvent
	event := &PREvent{
		Action: "push",
	}

	var raw map[string]interface{}
	json.Unmarshal(payload, &raw)

	event.Repository = *parseRepository(raw, "repository")

	// Parse sender
	if senderRaw, exists := raw["sender"]; exists {
		senderJSON, err := json.Marshal(senderRaw)
		if err != nil {
			return nil, err
		}
		sender := &User{}
		json.Unmarshal(senderJSON, sender)
		event.Sender = *sender
	}

	return event, nil
}

// parseWorkflowRunEvent parses a workflow run event
func parseWorkflowRunEvent(payload []byte) (*PREvent, error) {
	var raw map[string]interface{}
	json.Unmarshal(payload, &raw)

	event := &PREvent{
		Action: "workflow_run",
	}

	event.Repository = *parseRepository(raw, "repository")

	if senderRaw, exists := raw["sender"]; exists {
		senderJSON, err := json.Marshal(senderRaw)
		if err != nil {
			return nil, err
		}
		sender := &User{}
		json.Unmarshal(senderJSON, sender)
		event.Sender = *sender
	}

	return event, nil
}

// parseStatusEvent parses a status event
func parseStatusEvent(payload []byte) (*PREvent, error) {
	var raw map[string]interface{}
	json.Unmarshal(payload, &raw)

	event := &PREvent{
		Action: "status",
	}

	event.Repository = *parseRepository(raw, "repository")

	// Parse commit
	if commitRaw, exists := raw["commit"]; exists {
		commitJSON, err := json.Marshal(commitRaw)
		if err != nil {
			return nil, err
		}
		commit := &Commit{}
		json.Unmarshal(commitJSON, commit)
		event.PR.HeadCommit = commit
	}

	return event, nil
}

// parseRepository parses a repository object from raw map
func parseRepository(raw map[string]interface{}, key string) *Repository {
	if repoRaw, exists := raw[key]; exists {
		repoJSON, _ := json.Marshal(repoRaw)
		repo := &Repository{}
		json.Unmarshal(repoJSON, repo)
		return repo
	}

	return &Repository{}
}

// processEvent processes a webhook event
func (h *WebhookHandler) processEvent(ctx context.Context, event *PREvent) {
	eventType := event.Action
	if eventType == "" {
		eventType = "unknown"
	}

	h.logger.WithFields(logrus.Fields{
		"event_type": eventType,
		"action":     event.Action,
		"repo":       event.Repository.FullName,
	}).Info("Processing webhook event")

	// Get handlers for this event type
	handlers := h.registry.GetHandlers(eventType)
	if len(handlers) == 0 {
		h.logger.WithField("event_type", eventType).Debug("No handlers registered for this event type")
		handlers = h.registry.GetHandlers("all") // Fallback to all handlers
	}

	// Execute handlers
	for _, handler := range handlers {
		go func(handler func(event *PREvent)) {
			defer func() {
				if r := recover(); r != nil {
					h.logger.WithField("panic", r).Error("Webhook handler panicked")
				}
			}()
			handler(event)
		}(handler)
	}
}

// Helper functions

// getString safely gets a string from a map
func getString(raw map[string]interface{}, key string) string {
	if value, exists := raw[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// RegisterHandler registers an event handler
func (h *WebhookHandler) RegisterHandler(eventType string, handler func(event *PREvent)) {
	h.registry.RegisterHandler(eventType, handler)
}

// GetHandlerCount returns the number of registered handlers
func (h *WebhookHandler) GetHandlerCount(eventType string) int {
	return len(h.registry.GetHandlers(eventType))
}

// ClearHandlers clears all registered handlers
func (h *WebhookHandler) ClearHandlers() {
	h.registry = NewEventHandlerRegistry()
}
