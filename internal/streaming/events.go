package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// EventType represents the type of event
type EventType string

const (
	// EventBuildStarted indicates a build has started
	EventBuildStarted EventType = "build_started"
	// EventBuildCompleted indicates a build has completed
	EventBuildCompleted EventType = "build_completed"
	// EventBuildFailed indicates a build has failed
	EventBuildFailed EventType = "build_failed"
	// EventBuildUnstable indicates a build is unstable
	EventBuildUnstable EventType = "build_unstable"
	// EventBuildAborted indicates a build has been aborted
	EventBuildAborted EventType = "build_aborted"
	// EventDeploymentStarted indicates a deployment has started
	EventDeploymentStarted EventType = "deployment_started"
	// EventDeploymentCompleted indicates a deployment has completed
	EventDeploymentCompleted EventType = "deployment_completed"
	// EventDeploymentFailed indicates a deployment has failed
	EventDeploymentFailed EventType = "deployment_failed"
	// EventDeploySuccess indicates a deployment succeeded
	EventDeploySuccess EventType = "deploy_success"
	// EventPipelineStarted indicates a pipeline has started
	EventPipelineStarted EventType = "pipeline_started"
	// EventPipelineCompleted indicates a pipeline has completed
	EventPipelineCompleted EventType = "pipeline_completed"
	// EventPipelineFailed indicates a pipeline has failed
	EventPipelineFailed EventType = "pipeline_failed"
	// EventWorkflowStarted indicates a workflow has started
	EventWorkflowStarted EventType = "workflow_started"
	// EventWorkflowCompleted indicates a workflow has completed
	EventWorkflowCompleted EventType = "workflow_completed"
	// EventWorkflowFailed indicates a workflow has failed
	EventWorkflowFailed EventType = "workflow_failed"
	// EventPush indicates a git push event
	EventPush EventType = "push"
	// EventPRCreated indicates a PR has been created
	EventPRCreated EventType = "pr_created"
	// EventPRUpdated indicates a PR has been updated
	EventPRUpdated EventType = "pr_updated"
	// EventPRMerged indicates a PR has been merged
	EventPRMerged EventType = "pr_merged"
	// EventTestStarted indicates tests have started
	EventTestStarted EventType = "test_started"
	// EventTestCompleted indicates tests have completed
	EventTestCompleted EventType = "test_completed"
	// EventTestFailed indicates tests have failed
	EventTestFailed EventType = "test_failed"
)

// Event represents a streaming event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	SourceID  string                 `json:"source_id"`
	Payload   map[string]interface{} `json:"payload"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Severity  string                 `json:"severity,omitempty"`
}

// EventSource represents the source of an event
type EventSource string

const (
	SourceGitHub    EventSource = "github"
	SourceJenkins   EventSource = "jenkins"
	SourceKubernetes EventSource = "kubernetes"
	SourceCI        EventSource = "ci"
	SourceCustom    EventSource = "custom"
)

// EventProducer produces events
type EventProducer struct {
	logger   *logrus.Logger
	brokers  []EventBroker
	mu       sync.RWMutex
}

// EventBroker handles event distribution
type EventBroker interface {
	Publish(ctx context.Context, event *Event) error
	Subscribe(eventType EventType, handler EventHandler) error
	Unsubscribe(eventType EventType, handler EventHandler) error
	Start(ctx context.Context) error
	Stop() error
}

// EventHandler handles incoming events
type EventHandler func(ctx context.Context, event *Event)

// InMemoryBroker is an in-memory event broker
type InMemoryBroker struct {
	logger    *logrus.Logger
	handlers  map[EventType][]EventHandler
	eventChan chan *Event
	cleanupChan chan chan struct{}
	config    BrokerConfig
}

// BrokerConfig contains broker configuration
type BrokerConfig struct {
	MaxBufferSize int
	BufferTimeout time.Duration
	EnableMetrics bool
}

// NewInMemoryBroker creates a new in-memory broker
func NewInMemoryBroker(logger *logrus.Logger, config BrokerConfig) *InMemoryBroker {
	if config.MaxBufferSize == 0 {
		config.MaxBufferSize = 1000
	}

	return &InMemoryBroker{
		logger:    logger.WithField("component", "event_broker"),
		handlers:  make(map[EventType][]EventHandler),
		eventChan: make(chan *Event, config.MaxBufferSize),
		cleanupChan: make(chan chan struct{}),
		config:    config,
	}
}

// Publish publishes an event
func (b *InMemoryBroker) Publish(ctx context.Context, event *Event) error {
	select {
	case b.eventChan <- event:
		b.logger.WithFields(logrus.Fields{
			"type":    event.Type,
			"source":  event.Source,
		}).Debug("Event published")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("publish context cancelled: %w", ctx.Err())
	default:
		return fmt.Errorf("event queue full")
	}
}

// Subscribe subscribes to an event type
func (b *InMemoryBroker) Subscribe(eventType EventType, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.logger.WithFields(logrus.Fields{
		"type":    eventType,
		"handler": fmt.Sprintf("%p", handler),
	}).Debug("Handler subscribed")

	return nil
}

// Unsubscribe unsubscribes from an event type
func (b *InMemoryBroker) Unsubscribe(eventType EventType, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers := b.handlers[eventType]
	for i, h := range handlers {
		if h == handler {
			b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			b.logger.WithFields(logrus.Fields{
				"type":    eventType,
				"handler": fmt.Sprintf("%p", handler),
			}).Debug("Handler unsubscribed")
			return nil
		}
	}

	return fmt.Errorf("handler not found for event type: %s", eventType)
}

// Start starts the broker
func (b *InMemoryBroker) Start(ctx context.Context) error {
	b.logger.Info("Event broker started")

	go b.processEvents(ctx)
	return nil
}

// Stop stops the broker
func (b *InMemoryBroker) Stop() error {
	b.logger.Info("Event broker stopped")
	return nil
}

// processEvents processes events from the queue
func (b *InMemoryBroker) processEvents(ctx context.Context) {
	for {
		select {
		case event := <-b.eventChan:
			b.processEvent(event)
		case <-ctx.Done():
			return
		}
	}
}

// processEvent processes a single event
func (b *InMemoryBroker) processEvent(event *Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	handlers := b.handlers[event.Type]
	if handlers == nil {
		handlers = b.handlers[EventType("*")] // Fallback to wildcard
	}

	for _, handler := range handlers {
		go func(handler EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					b.logger.WithFields(logrus.Fields{
						"event_type": event.Type,
						"panic":      r,
					}).Error("Handler panicked")
				}
			}()
			handler(context.Background(), event)
		}(handler)
	}
}

// GetHandlerCount returns the number of handlers for an event type
func (b *InMemoryBroker) GetHandlerCount(eventType EventType) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[eventType])
}

// EventProducerOptions contains producer options
type EventProducerOptions struct {
	Broker    EventBroker
	Source    EventSource
	MaxRetries int
	RetryDelay time.Duration
}

// NewEventProducer creates a new event producer
func NewEventProducer(logger *logrus.Logger, options EventProducerOptions) *EventProducer {
	if options.MaxRetries == 0 {
		options.MaxRetries = 3
	}

	if options.RetryDelay == 0 {
		options.RetryDelay = 1 * time.Second
	}

	return &EventProducer{
		logger:   logger.WithField("component", "event_producer"),
		brokers:  []EventBroker{options.Broker},
	}
}

// GenerateEventID generates a unique event ID
func GenerateEventID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%10000)
}

// CreateEvent creates a new event
func (p *EventProducer) CreateEvent(eventType EventType, source EventSource, sourceID string, payload map[string]interface{}) *Event {
	return &Event{
		ID:        GenerateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Source:    string(source),
		SourceID:  sourceID,
		Payload:   payload,
		TraceID:   p.generateTraceID(),
		Severity:  "info",
	}
}

// generateTraceID generates a trace ID for distributed tracing
func (p *EventProducer) generateTraceID() string {
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}

// PublishEvent publishes an event to all brokers
func (p *EventProducer) PublishEvent(ctx context.Context, event *Event) error {
	var lastErr error

	for _, broker := range p.brokers {
		if err := broker.Publish(ctx, event); err != nil {
			lastErr = err
			p.logger.WithFields(logrus.Fields{
				"event_id":  event.ID,
				"broker":    fmt.Sprintf("%p", broker),
			}).WithError(err).Warn("Failed to publish event")
		}
	}

	if lastErr != nil {
		return lastErr
	}

	return nil
}

// PublishEventWithRetry publishes an event with retry
func (p *EventProducer) PublishEventWithRetry(ctx context.Context, event *Event, retries int) error {
	for i := 0; i <= retries; i++ {
		err := p.PublishEvent(ctx, event)
		if err == nil {
			return nil
		}

		if i < retries {
			time.Sleep(1 * time.Second)
		}
	}

	return fmt.Errorf("failed to publish event after %d retries", retries)
}

// SubscribeToEvent subscribes to an event type
func (p *EventProducer) SubscribeToEvent(eventType EventType, handler EventHandler) error {
	for _, broker := range p.brokers {
		if err := broker.Subscribe(eventType, handler); err != nil {
			return err
		}
	}
	return nil
}

// UnsubscribeFromEvent unsubscribes from an event type
func (p *EventProducer) UnsubscribeFromEvent(eventType EventType, handler EventHandler) error {
	for _, broker := range p.brokers {
		if err := broker.Unsubscribe(eventType, handler); err != nil {
			return err
		}
	}
	return nil
}

// StartAllBrokers starts all event brokers
func (p *EventProducer) StartAllBrokers(ctx context.Context) error {
	for _, broker := range p.brokers {
		if err := broker.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// StopAllBrokers stops all event brokers
func (p *EventProducer) StopAllBrokers() error {
	for _, broker := range p.brokers {
		if err := broker.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// GetBrokerCount returns the number of registered brokers
func (p *EventProducer) GetBrokerCount() int {
	return len(p.brokers)
}

// EventFactory provides factory methods for creating events
type EventFactory struct {
	logger *logrus.Logger
}

// NewEventFactory creates a new event factory
func NewEventFactory(logger *logrus.Logger) *EventFactory {
	return &EventFactory{
		logger: logger.WithField("component", "event_factory"),
	}
}

// NewBuildStartedEvent creates a build started event
func (f *EventFactory) NewBuildStartedEvent(buildID string, jobName string, triggeredBy string) *Event {
	return f.CreateEvent(EventBuildStarted, SourceCI, buildID, map[string]interface{}{
		"job_name":    jobName,
		"triggered_by": triggeredBy,
		"build_id":    buildID,
	})
}

// NewBuildCompletedEvent creates a build completed event
func (f *EventFactory) NewBuildCompletedEvent(buildID string, jobName string, result string, duration int) *Event {
	return f.CreateEvent(EventBuildCompleted, SourceCI, buildID, map[string]interface{}{
		"job_name":   jobName,
		"result":     result,
		"duration":   duration,
		"build_id":   buildID,
	})
}

// NewBuildFailedEvent creates a build failed event
func (f *EventFactory) NewBuildFailedEvent(buildID string, jobName string, failureReason string, errorInfo map[string]interface{}) *Event {
	event := f.CreateEvent(EventBuildFailed, SourceCI, buildID, map[string]interface{}{
		"job_name":      jobName,
		"failure_reason": failureReason,
		"error_info":    errorInfo,
		"build_id":      buildID,
	})
	event.Severity = "error"
	return event
}

// NewDeploymentStartedEvent creates a deployment started event
func (f *EventFactory) NewDeploymentStartedEvent(deploymentID string, environment string, service string) *Event {
	return f.CreateEvent(EventDeploymentStarted, SourceCI, deploymentID, map[string]interface{}{
		"environment": environment,
		"service":     service,
		"deployment_id": deploymentID,
	})
}

// NewWorkflowStartedEvent creates a workflow started event
func (f *EventFactory) NewWorkflowStartedEvent(workflowID string, workflowName string, eventType string) *Event {
	return f.CreateEvent(EventWorkflowStarted, SourceGitHub, workflowID, map[string]interface{}{
		"workflow_name": workflowName,
		"event_type":    eventType,
		"workflow_id":   workflowID,
	})
}

// NewWorkflowCompletedEvent creates a workflow completed event
func (f *EventFactory) NewWorkflowCompletedEvent(workflowID string, workflowName string, conclusion string, duration int) *Event {
	return f.CreateEvent(EventWorkflowCompleted, SourceGitHub, workflowID, map[string]interface{}{
		"workflow_name": workflowName,
		"conclusion":    conclusion,
		"duration":      duration,
		"workflow_id":   workflowID,
	})
}

// NewPRCreatedEvent creates a PR created event
func (f *EventFactory) NewPRCreatedEvent(prID int, repo string, branch string, author string) *Event {
	return f.CreateEvent(EventPRCreated, SourceGitHub, fmt.Sprintf("pr-%d", prID), map[string]interface{}{
		"pr_id":     prID,
		"repo":      repo,
		"branch":    branch,
		"author":    author,
	})
}

// NewPRMergedEvent creates a PR merged event
func (f *EventFactory) NewPRMergedEvent(prID int, repo string, mergedBy string) *Event {
	return f.CreateEvent(EventPRMerged, SourceGitHub, fmt.Sprintf("pr-%d", prID), map[string]interface{}{
		"pr_id":     prID,
		"repo":      repo,
		"merged_by": mergedBy,
	})
}

// NewTestStartedEvent creates a test started event
func (f *EventFactory) NewTestStartedEvent(testSuite string, testCount int) *Event {
	return f.CreateEvent(EventTestStarted, SourceCI, testSuite, map[string]interface{}{
		"test_suite":   testSuite,
		"test_count":   testCount,
	})
}

// NewTestFailedEvent creates a test failed event
func (f *EventFactory) NewTestFailedEvent(testSuite string, failedTests []string) *Event {
	event := f.CreateEvent(EventTestFailed, SourceCI, testSuite, map[string]interface{}{
		"test_suite":    testSuite,
		"failed_tests":  failedTests,
	})
	event.Severity = "error"
	return event
}

// CreateEvent creates a new event with given type
func (f *EventFactory) CreateEvent(eventType EventType, source EventSource, sourceID string, payload map[string]interface{}) *Event {
	return &Event{
		ID:        GenerateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Source:    string(source),
		SourceID:  sourceID,
		Payload:   payload,
		TraceID:   GenerateEventID(),
		Severity:  "info",
	}
}

// EventValidator validates events
type EventValidator struct {
	requiredFields map[EventType][]string
}

// NewEventValidator creates a new event validator
func NewEventValidator() *EventValidator {
	return &EventValidator{
		requiredFields: map[EventType][]string{
			EventBuildStarted:   {"build_id", "job_name"},
			EventBuildCompleted: {"build_id", "job_name", "result"},
			EventBuildFailed:    {"build_id", "job_name", "failure_reason"},
			EventDeploymentStarted: {"deployment_id", "environment"},
			EventWorkflowStarted: {"workflow_id", "workflow_name"},
			EventWorkflowCompleted: {"workflow_id", "workflow_name"},
			EventPRCreated: {"pr_id", "repo"},
			EventPRMerged: {"pr_id", "repo"},
			EventTestStarted:   {"test_suite", "test_count"},
		},
	}
}

// Validate validates an event
func (v *EventValidator) Validate(event *Event) bool {
	requiredFields := v.requiredFields[event.Type]
	for _, field := range requiredFields {
		if _, exists := event.Payload[field]; !exists {
			return false
		}
	}
	return true
}

// GetMissingFields returns missing required fields
func (v *EventValidator) GetMissingFields(event *Event) []string {
	missing := []string{}
	requiredFields := v.requiredFields[event.Type]

	for _, field := range requiredFields {
		if _, exists := event.Payload[field]; !exists {
			missing = append(missing, field)
		}
	}

	return missing
}

// EventSerializer serializes events
type EventSerializer struct{}

// NewEventSerializer creates a new event serializer
func NewEventSerializer() *EventSerializer {
	return &EventSerializer{}
}

// Serialize serializes an event to JSON
func (s *EventSerializer) Serialize(event *Event) ([]byte, error) {
	return json.Marshal(event)
}

// Deserialize deserializes an event from JSON
func (s *EventSerializer) Deserialize(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// EventFilter filters events
type EventFilter struct {
	EventType  EventType
	Source     string
	SourceID   string
	AfterTime  time.Time
	BeforeTime time.Time
}

// Filter filters events
func (f *EventFilter) Filter(events []*Event) []*Event {
	var filtered []*Event

	for _, event := range events {
		if !f.matches(event) {
			continue
		}
		filtered = append(filtered, event)
	}

	return filtered
}

func (f *EventFilter) matches(event *Event) bool {
	if f.EventType != "" && event.Type != f.EventType {
		return false
	}

	if f.Source != "" && event.Source != f.Source {
		return false
	}

	if f.SourceID != "" && event.SourceID != f.SourceID {
		return false
	}

	if !f.AfterTime.IsZero() && event.Timestamp.Before(f.AfterTime) {
		return false
	}

	if !f.BeforeTime.IsZero() && event.Timestamp.After(f.BeforeTime) {
		return false
	}

	return true
}
