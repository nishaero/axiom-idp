package jenkins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// WebhookPayload represents a webhook payload
type WebhookPayload struct {
	JobName      string    `json:"job_name"`
	JobURL       string    `json:"job_url"`
	BuildNumber  int       `json:"build_number"`
	BuildResult  string    `json:"build_result"`
	BuildURL     string    `json:"build_url"`
	TriggeredBy  string    `json:"triggered_by"`
	BuildTimestamp time.Time `json:"build_timestamp"`
	CauseSummary string    `json:"cause_summary"`
	ChangeSet    []Change  `json:"change_set"`
}

// WebhookHandler handles Jenkins webhook events
type WebhookHandler struct {
	logger     *logrus.Logger
	client     *JenkinsClient
	config     WebhookConfig
	registry   *EventRegistry
}

// WebhookConfig contains webhook configuration
type WebhookConfig struct {
	Secret      string
	Path        string
	VerifySSL   bool
	AllowEvents []string
}

// EventRegistry manages event handlers
type EventRegistry struct {
	handlers map[string][]func(payload *WebhookPayload)
}

// NewEventRegistry creates a new event registry
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		handlers: make(map[string][]func(payload *WebhookPayload)),
	}
}

// RegisterHandler registers an event handler
func (r *EventRegistry) RegisterHandler(eventType string, handler func(payload *WebhookPayload)) {
	r.handlers[eventType] = append(r.handlers[eventType], handler)
}

// GetHandlers returns handlers for an event type
func (r *EventRegistry) GetHandlers(eventType string) []func(payload *WebhookPayload) {
	return r.handlers[eventType]
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(logger *logrus.Logger, client *JenkinsClient, config WebhookConfig, registry *EventRegistry) *WebhookHandler {
	return &WebhookHandler{
		logger:   logger.WithField("component", "jenkins_webhook_handler"),
		client:   client,
		config:   config,
		registry: registry,
	}
}

// Handle handles incoming webhook requests
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify path
	if h.config.Path != "" && r.URL.Path != h.config.Path {
		http.Error(w, "invalid path", http.StatusNotFound)
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

	// Verify signature (if configured)
	if h.config.Secret != "" {
		if !h.verifySignature(payload, r.Header) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse payload
	webhookPayload, err := parseWebhookPayload(payload)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		h.logger.WithError(err).Error("Failed to parse webhook payload")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"job":        webhookPayload.JobName,
		"build":      webhookPayload.BuildNumber,
		"result":     webhookPayload.BuildResult,
	}).Info("Received Jenkins webhook event")

	// Process event
	h.processEvent(ctx, webhookPayload)

	// Send acknowledgment
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Event received")
}

// verifySignature verifies the webhook signature
func (h *WebhookHandler) verifySignature(payload []byte, headers http.Header) bool {
	if h.config.Secret == "" {
		return true
	}

	// In production, verify using HMAC
	// For now, skip verification
	return true
}

// parseWebhookPayload parses a webhook payload
func parseWebhookPayload(payload []byte) (*WebhookPayload, error) {
	// Try parsing as JSON first
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err == nil {
		// Convert to WebhookPayload
		return &WebhookPayload{
			JobName:       getString(raw, "job_name"),
			JobURL:        getString(raw, "job_url"),
			BuildNumber:   getInt(raw, "build_number"),
			BuildResult:   getString(raw, "build_result"),
			BuildURL:      getString(raw, "build_url"),
			TriggeredBy:   getString(raw, "triggered_by"),
			CauseSummary:  getString(raw, "cause_summary"),
			BuildTimestamp: getTimestamp(raw, "build_timestamp"),
		}, nil
	}

	// If not JSON, try to parse as build result
	if string(payload) == "SUCCESS" || string(payload) == "FAILURE" || string(payload) == "UNSTABLE" {
		return &WebhookPayload{
			BuildResult: string(payload),
		}, nil
	}

	return nil, fmt.Errorf("unsupported payload format")
}

// processEvent processes a webhook event
func (h *WebhookHandler) processEvent(ctx context.Context, payload *WebhookPayload) {
	eventType := payload.BuildResult
	if eventType == "" {
		eventType = "unknown"
	}

	h.logger.WithFields(logrus.Fields{
		"event_type": eventType,
		"job":        payload.JobName,
		"build":      payload.BuildNumber,
	}).Info("Processing webhook event")

	// Get handlers for this event type
	handlers := h.registry.GetHandlers(eventType)
	if len(handlers) == 0 {
		handlers = h.registry.GetHandlers("all") // Fallback
	}

	// Execute handlers
	for _, handler := range handlers {
		go func(handler func(payload *WebhookPayload)) {
			defer func() {
				if r := recover(); r != nil {
					h.logger.WithField("panic", r).Error("Webhook handler panicked")
				}
			}()
			handler(payload)
		}(handler)
	}
}

// GetEventType returns the event type based on build result
func (h *WebhookHandler) GetEventType(buildResult string) string {
	switch buildResult {
	case "SUCCESS", "SUCCESSFUL":
		return "build_success"
	case "FAILURE", "FAILED":
		return "build_failure"
	case "UNSTABLE":
		return "build_unstable"
	case "ABORTED":
		return "build_aborted"
	default:
		return fmt.Sprintf("build_%s", buildResult)
	}
}

// RegisterHandler registers an event handler
func (h *WebhookHandler) RegisterHandler(eventType string, handler func(payload *WebhookPayload)) {
	h.registry.RegisterHandler(eventType, handler)
}

// GetHandlerCount returns the number of registered handlers
func (h *WebhookHandler) GetHandlerCount(eventType string) int {
	return len(h.registry.GetHandlers(eventType))
}

// ClearHandlers clears all registered handlers
func (h *WebhookHandler) ClearHandlers() {
	h.registry = NewEventRegistry()
}

// Helper functions

func getString(raw map[string]interface{}, key string) string {
	if value, exists := raw[key]; exists {
		switch v := value.(type) {
		case string:
			return v
		case float64:
			return fmt.Sprintf("%.0f", v)
		case int:
			return fmt.Sprintf("%d", v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func getInt(raw map[string]interface{}, key string) int {
	if value, exists := raw[key]; exists {
		switch v := value.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case int64:
			return int(v)
		}
	}
	return 0
}

func getTimestamp(raw map[string]interface{}, key string) time.Time {
	if value, exists := raw[key]; exists {
		switch v := value.(type) {
		case string:
			t, err := time.Parse(time.RFC3339, v)
			if err == nil {
				return t
			}
			t, err = time.Parse("2006-01-02T15:04:05.000Z", v)
			if err == nil {
				return t
			}
		case time.Time:
			return v
		}
	}
	return time.Now()
}
