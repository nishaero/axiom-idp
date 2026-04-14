package gitlab

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// WebhookHandler handles GitLab webhook events
type WebhookHandler struct {
	logger   *logrus.Logger
	client   *GitLabClient
	secret   string
	path     string
	registry *EventRegistry
	config   *WebhookConfig
}

// EventRegistry manages event handlers for webhooks
type EventRegistry struct {
	handlers map[string][]func(event *WebhookEvent)
}

// NewEventRegistry creates a new event registry
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		handlers: make(map[string][]func(event *WebhookEvent)),
	}
}

// RegisterHandler registers an event handler for a specific event type
func (r *EventRegistry) RegisterHandler(eventType string, handler func(event *WebhookEvent)) {
	r.handlers[eventType] = append(r.handlers[eventType], handler)
}

// GetHandlers returns all handlers for an event type
func (r *EventRegistry) GetHandlers(eventType string) []func(event *WebhookEvent) {
	return r.handlers[eventType]
}

// NewWebhookHandler creates a new GitLab webhook handler
func NewWebhookHandler(logger *logrus.Logger, client *GitLabClient, secret string, registry *EventRegistry) *WebhookHandler {
	return &WebhookHandler{
		logger:   logger,
		client:   client,
		secret:   secret,
		registry: registry,
		config: &WebhookConfig{
			Path:           "/api/v1/ci/gitlab/webhook",
			VerifySSL:      true,
			EnableAuditLog: true,
		},
	}
}

// SetConfig sets the webhook configuration
func (h *WebhookHandler) SetConfig(config WebhookConfig) {
	h.config = &config
	if config.Path != "" {
		h.path = config.Path
	}
	if config.Secret != "" {
		h.secret = config.Secret
	}
	if len(config.AllowedEvents) > 0 {
		h.config.AllowedEvents = config.AllowedEvents
	}
}

// Handle handles incoming GitLab webhook requests
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Log request details
	h.logger.WithFields(logrus.Fields{
		"method":    r.Method,
		"path":      r.URL.Path,
		"event":     r.Header.Get("X-Gitlab-Event"),
		"timestamp": time.Now().Format(time.RFC3339),
	}).Debug("Received webhook request")

	// Verify path if configured
	if h.path != "" && r.URL.Path != h.path && r.URL.Path != h.path+"/" {
		http.Error(w, "invalid path", http.StatusNotFound)
		h.logger.WithFields(logrus.Fields{
			"expected": h.path,
			"received": r.URL.Path,
		}).Warn("Invalid webhook path")
		return
	}

	// Get event type from header
	eventType := r.Header.Get("X-Gitlab-Event")
	if eventType == "" {
		http.Error(w, "missing X-Gitlab-Event header", http.StatusBadRequest)
		h.logger.Warn("Missing X-Gitlab-Event header")
		return
	}

	// Check if event type is allowed
	if !h.isEventAllowed(eventType) {
		http.Error(w, "event type not allowed", http.StatusForbidden)
		h.logger.WithField("event_type", eventType).Warn("Event type not allowed")
		return
	}

	// Verify signature
	signature := r.Header.Get("X-Gitlab-Token")
	if !h.verifySignature(r, signature) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		h.logger.WithField("event_type", eventType).Warn("Invalid webhook signature")
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
	event, err := h.parseWebhookEvent(payload, eventType)
	if err != nil {
		http.Error(w, "invalid event", http.StatusBadRequest)
		h.logger.WithError(err).Error("Failed to parse webhook event")
		return
	}

	// Log event details
	h.logger.WithFields(logrus.Fields{
		"event_type": eventType,
		"project":    event.Project.FullName,
		"ref":        event.Ref,
		"author":     event.Author,
	}).Info("Received GitLab webhook event")

	// Audit log
	if h.config.EnableAuditLog {
		h.auditLog(eventType, event)
	}

	// Process event
	h.processEvent(ctx, event, eventType)

	// Send acknowledgment
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Event received")
}

// isEventAllowed checks if an event type is allowed
func (h *WebhookHandler) isEventAllowed(eventType string) bool {
	if len(h.config.AllowedEvents) == 0 {
		return true
	}

	for _, allowed := range h.config.AllowedEvents {
		if allowed == eventType {
			return true
		}
	}

	return false
}

// verifySignature verifies the GitLab webhook signature
func (h *WebhookHandler) verifySignature(r *http.Request, signature string) bool {
	if h.secret == "" {
		// No secret configured, skip verification (development mode)
		h.logger.Warn("No webhook secret configured, skipping signature verification")
		return true
	}

	if signature == "" {
		h.logger.Warn("Missing signature header")
		return false
	}

	// GitLab uses a shared secret token (not HMAC for webhook tokens)
	// The token should match the secret configured in GitLab webhook settings
	return signature == h.secret
}

// parseWebhookEvent parses a webhook payload into a WebhookEvent
func (h *WebhookHandler) parseWebhookEvent(payload []byte, eventType string) (*WebhookEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	event := &WebhookEvent{
		EventType: eventType,
	}

	// Parse common fields
	event.Project = *parseProject(raw, "project")
	event.Ref = getString(raw, "ref")
	event.BeforeSha = getString(raw, "before_sha")
	event.AfterSha = getString(raw, "after_sha")

	// Parse author
	if authorRaw, exists := raw["author"]; exists {
		authorJSON, _ := json.Marshal(authorRaw)
		var author User
		json.Unmarshal(authorJSON, &author)
		event.Author = &author
	}

	// Parse event name
	if eventName, exists := raw["event_name"]; exists {
		if name, ok := eventName.(string); ok {
			event.EventName = name
		}
	}

	// Parse event type-specific data
	switch eventType {
	case "Push Hook":
		// Parse commits
		if commitsRaw, exists := raw["commits"]; exists {
			var commits []Commit
			if commitsJSON, err := json.Marshal(commitsRaw); err == nil {
				json.Unmarshal(commitsJSON, &commits)
			}
			event.Commits = commits
		}

		// Parse labels
		if labelsRaw, exists := raw["labels"]; exists {
			var labels []map[string]interface{}
			if labelsJSON, err := json.Marshal(labelsRaw); err == nil {
				json.Unmarshal(labelsJSON, &labels)
			}
			event.Labels = labels
		}

	case "Pipeline Hook", "Pipeline Status Hook":
		// Parse pipeline
		if pipelineRaw, exists := raw["pipeline"]; exists {
			pipelineJSON, _ := json.Marshal(pipelineRaw)
			var pipeline Pipeline
			if err := json.Unmarshal(pipelineJSON, &pipeline); err == nil {
				event.Pipeline = &pipeline
			}
		}

		// Parse build (for compatibility)
		if buildRaw, exists := raw["build"]; exists {
			buildJSON, _ := json.Marshal(buildRaw)
			var build Job
			if err := json.Unmarshal(buildJSON, &build); err == nil {
				event.Job = &build
			}
		}

	case "Job Hook":
		// Parse job
		if jobRaw, exists := raw["build"]; exists {
			jobJSON, _ := json.Marshal(jobRaw)
			var job Job
			if err := json.Unmarshal(jobJSON, &job); err == nil {
				event.Job = &job
			}
		}

		// Parse runner
		if runnerRaw, exists := raw["runner"]; exists {
			runnerJSON, _ := json.Marshal(runnerRaw)
			var runner Runner
			if err := json.Unmarshal(runnerJSON, &runner); err == nil {
				event.Runner = &runner
			}
		}

	case "Merge Request Hook":
		// Parse merge request
		if mrRaw, exists := raw["object_attributes"]; exists {
			mrJSON, _ := json.Marshal(mrRaw)
			var mr MergeRequest
			if err := json.Unmarshal(mrJSON, &mr); err == nil {
				event.MergeRequest = &mr
			}
		}

		// Also try parsing from object_event (newer GitLab versions)
		if mrRaw, exists := raw["merge_request"]; exists {
			mrJSON, _ := json.Marshal(mrRaw)
			var mr MergeRequest
			if err := json.Unmarshal(mrJSON, &mr); err == nil {
				event.MergeRequest = &mr
			}
		}

		// Parse labels if present
		if labelsRaw, exists := raw["labels"]; exists {
			var labels []map[string]interface{}
			json.Unmarshal([]byte(fmt.Sprint(labelsRaw)), &labels)
			event.Labels = labels
		}

	default:
		h.logger.WithField("event_type", eventType).Debug("Unknown event type, parsing as generic")
	}

	return event, nil
}

// parseProject parses a project object from raw map
func parseProject(raw map[string]interface{}, key string) *Project {
	if projectRaw, exists := raw[key]; exists {
		projectJSON, _ := json.Marshal(projectRaw)
		project := &Project{}
		json.Unmarshal(projectJSON, project)
		return project
	}

	return &Project{}
}

// getString safely gets a string from a map
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

// auditLog logs the webhook event for audit purposes
func (h *WebhookHandler) auditLog(eventType string, event *WebhookEvent) {
	auditData := map[string]interface{}{
		"timestamp":    time.Now().UnixNano(),
		"event_type":   eventType,
		"project_id":   event.Project.ID,
		"project_name": event.Project.Name,
		"ref":          event.Ref,
		"event_name":   event.EventName,
	}

	h.logger.WithFields(logrus.Fields{
		"audit": auditData,
	}).Info("Webhook audit log")
}

// processEvent processes a webhook event by dispatching to registered handlers
func (h *WebhookHandler) processEvent(ctx context.Context, event *WebhookEvent, eventType string) {
	// Determine event category for handler matching
	eventCategory := h.getEventCategory(eventType)

	h.logger.WithFields(logrus.Fields{
		"event_category": eventCategory,
		"event_type":     eventType,
		"project":        event.Project.FullName,
	}).Info("Processing webhook event")

	// Get handlers for this event category
	handlers := h.registry.GetHandlers(eventCategory)

	// Also try the specific event type
	if len(handlers) == 0 {
		handlers = h.registry.GetHandlers(eventType)
	}

	// Fallback to all handlers
	if len(handlers) == 0 {
		handlers = h.registry.GetHandlers("all")
	}

	// Execute handlers
	for _, handler := range handlers {
		go func(handler func(event *WebhookEvent)) {
			defer func() {
				if r := recover(); r != nil {
					h.logger.WithField("panic", r).Error("Webhook handler panicked")
				}
			}()
			handler(event)
		}(handler)
	}
}

// getEventCategory maps GitLab event types to handler categories
func (h *WebhookHandler) getEventCategory(eventType string) string {
	switch eventType {
	case "Push Hook":
		return "push"
	case "Pipeline Hook", "Pipeline Status Hook":
		return "pipeline"
	case "Job Hook":
		return "job"
	case "Merge Request Hook":
		return "merge_request"
	case "Tag Push Hook":
		return "tag"
	default:
		return eventType
	}
}

// GetEventType returns the normalized event type
func (h *WebhookHandler) GetEventType(eventType string) string {
	return h.getEventCategory(eventType)
}

// RegisterHandler registers an event handler for a specific event type
func (h *WebhookHandler) RegisterHandler(eventType string, handler func(event *WebhookEvent)) {
	h.registry.RegisterHandler(eventType, handler)
	h.logger.WithField("event_type", eventType).Info("Webhook handler registered")
}

// GetHandlerCount returns the number of registered handlers
func (h *WebhookHandler) GetHandlerCount(eventType string) int {
	return len(h.registry.GetHandlers(eventType))
}

// ClearHandlers clears all registered handlers
func (h *WebhookHandler) ClearHandlers() {
	h.registry = NewEventRegistry()
	h.logger.Info("Webhook handlers cleared")
}

// GetEventCategory returns the category for a GitLab event type
func (h *WebhookHandler) GetEventCategory(eventType string) string {
	return h.getEventCategory(eventType)
}

// PushEventHandler handles push events
func (h *WebhookHandler) PushEventHandler(event *WebhookEvent) {
	h.logger.WithFields(logrus.Fields{
		"project": event.Project.FullName,
		"ref":     event.Ref,
		"commits": len(event.Commits),
		"author":  event.Author,
	}).Info("Processing push event")

	// Integration with service discovery for new deployments
	// This would connect to the CI orchestration system
}

// PipelineEventHandler handles pipeline events
func (h *WebhookHandler) PipelineEventHandler(event *WebhookEvent) {
	if event.Pipeline == nil {
		return
	}

	h.logger.WithFields(logrus.Fields{
		"pipeline_id": event.Pipeline.ID,
		"status":      event.Pipeline.Status,
		"ref":         event.Pipeline.Ref,
		"sha":         event.Pipeline.Sha,
		"duration":    event.Pipeline.Duration,
	}).Info("Processing pipeline event")

	// Stream pipeline status to CI orchestration
	// Track pipeline execution metrics
}

// JobEventHandler handles job events
func (h *WebhookHandler) JobEventHandler(event *WebhookEvent) {
	if event.Job == nil {
		return
	}

	h.logger.WithFields(logrus.Fields{
		"job_id":   event.Job.ID,
		"job_name": event.Job.Name,
		"status":   event.Job.Status,
		"stage":    event.Job.Stage,
		"duration": event.Job.Duration,
	}).Info("Processing job event")

	// Stream job status to CI orchestration
	// Collect job metrics
}

// MergeRequestEventHandler handles merge request events
func (h *WebhookHandler) MergeRequestEventHandler(event *WebhookEvent) {
	if event.MergeRequest == nil {
		return
	}

	h.logger.WithFields(logrus.Fields{
		"mr_id":  event.MergeRequest.ID,
		"mr_iid": event.MergeRequest.IID,
		"state":  event.MergeRequest.State,
		"source": event.MergeRequest.SourceBranch,
		"target": event.MergeRequest.TargetBranch,
	}).Info("Processing merge request event")

	// Check if MR has associated pipelines
	// Update service catalog with new deployments
}

// VerifySignature verifies a webhook signature
func (h *WebhookHandler) VerifySignature(payload []byte, signature string) bool {
	return h.verifySignature(&http.Request{Header: http.Header{"X-Gitlab-Token": []string{signature}}}, signature)
}

// HMACSHA256 computes HMAC-SHA256 signature for webhook verification
func HMACSHA256(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return fmt.Sprintf("sha256=0%x", h.Sum(nil))
}

// CheckEventAllowed checks if an event type is allowed
func (h *WebhookHandler) CheckEventAllowed(eventType string) bool {
	return h.isEventAllowed(eventType)
}

// GetAllowedEvents returns the list of allowed events
func (h *WebhookHandler) GetAllowedEvents() []string {
	if len(h.config.AllowedEvents) == 0 {
		return []string{"all"}
	}
	return h.config.AllowedEvents
}

// SetAllowedEvents sets the list of allowed events
func (h *WebhookHandler) SetAllowedEvents(events []string) {
	h.config.AllowedEvents = events
}

// SetupHandler returns the HTTP handler for the webhook endpoint
func (h *WebhookHandler) SetupHandler() http.Handler {
	return http.HandlerFunc(h.Handle)
}

// HealthCheck returns the health status of the webhook handler
func (h *WebhookHandler) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status":        "healthy",
		"handler_count": h.GetHandlerCount("all"),
		"has_secret":    h.secret != "",
		"has_client":    h.client != nil,
		"verify_ssl":    h.config.VerifySSL,
	}
}

// GetClient returns the GitLab client
func (h *WebhookHandler) GetClient() *GitLabClient {
	return h.client
}

// SetClient sets the GitLab client
func (h *WebhookHandler) SetClient(client *GitLabClient) {
	h.client = client
}

// GetPath returns the webhook path
func (h *WebhookHandler) GetPath() string {
	return h.path
}

// SetPath sets the webhook path
func (h *WebhookHandler) SetPath(path string) {
	h.path = path
}

// RegisterPushHandler registers a handler for push events
func (h *WebhookHandler) RegisterPushHandler(handler func(event *WebhookEvent)) {
	h.RegisterHandler("push", handler)
}

// RegisterPipelineHandler registers a handler for pipeline events
func (h *WebhookHandler) RegisterPipelineHandler(handler func(event *WebhookEvent)) {
	h.RegisterHandler("pipeline", handler)
}

// RegisterJobHandler registers a handler for job events
func (h *WebhookHandler) RegisterJobHandler(handler func(event *WebhookEvent)) {
	h.RegisterHandler("job", handler)
}

// RegisterMergeRequestHandler registers a handler for merge request events
func (h *WebhookHandler) RegisterMergeRequestHandler(handler func(event *WebhookEvent)) {
	h.RegisterHandler("merge_request", handler)
}

// GetHandlerType returns the handler type for a GitLab event
func (h *WebhookHandler) GetHandlerType(eventType string) string {
	return h.getEventCategory(eventType)
}

// ParseWebhookEvent parses a webhook event (public method for testing)
func (h *WebhookHandler) ParseWebhookEvent(payload []byte, eventType string) (*WebhookEvent, error) {
	return h.parseWebhookEvent(payload, eventType)
}

// SetSecret sets the webhook secret
func (h *WebhookHandler) SetSecret(secret string) {
	h.secret = secret
}

// GetSecret returns the webhook secret
func (h *WebhookHandler) GetSecret() string {
	return h.secret
}

// SetEnableAuditLog enables or disables audit logging
func (h *WebhookHandler) SetEnableAuditLog(enable bool) {
	h.config.EnableAuditLog = enable
}

// GetEnableAuditLog returns whether audit logging is enabled
func (h *WebhookHandler) GetEnableAuditLog() bool {
	return h.config.EnableAuditLog
}

// SetVerifySSL enables or disables SSL verification
func (h *WebhookHandler) SetVerifySSL(verify bool) {
	h.config.VerifySSL = verify
}

// GetVerifySSL returns whether SSL verification is enabled
func (h *WebhookHandler) GetVerifySSL() bool {
	return h.config.VerifySSL
}

// ProcessPushEvent processes a push event
func (h *WebhookHandler) ProcessPushEvent(event *WebhookEvent) {
	h.PushEventHandler(event)
}

// ProcessPipelineEvent processes a pipeline event
func (h *WebhookHandler) ProcessPipelineEvent(event *WebhookEvent) {
	h.PipelineEventHandler(event)
}

// ProcessJobEvent processes a job event
func (h *WebhookHandler) ProcessJobEvent(event *WebhookEvent) {
	h.JobEventHandler(event)
}

// ProcessMergeRequestEvent processes a merge request event
func (h *WebhookHandler) ProcessMergeRequestEvent(event *WebhookEvent) {
	h.MergeRequestEventHandler(event)
}

// StreamEvent streams a webhook event to subscribers
func (h *WebhookHandler) StreamEvent(eventType string, event *WebhookEvent) {
	h.logger.WithField("event_type", eventType).Debug("Streaming event")
	h.processEvent(nil, event, eventType)
}

// GetEventMetrics returns metrics about processed events
func (h *WebhookHandler) GetEventMetrics() map[string]int {
	metrics := make(map[string]int)
	for eventType, handlers := range h.registry.handlers {
		metrics[eventType] = len(handlers)
	}
	return metrics
}

// String returns a string representation of the webhook handler
func (h *WebhookHandler) String() string {
	return fmt.Sprintf("GitLabWebhookHandler{path=%s, secret=%v, handlers=%d}",
		h.path, h.secret != "", h.GetHandlerCount("all"))
}
