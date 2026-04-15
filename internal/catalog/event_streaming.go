//go:build ignore

package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RedisEventStream provides event streaming using Redis streams
type RedisEventStream struct {
	mu          sync.RWMutex
	client      *redis.Client
	config      *RedisConfig
	logger      *logrus.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	subscribers map[string][]chan *Event
	streamName  string
	initialized bool
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host            string
	Port            int
	Password        string
	DB              int
	EnableStreaming bool
	Prefix          string
	StreamName      string
	ConsumerGroup   string
	ConsumerName    string
	MaxStreamLength int64
	PollInterval    time.Duration
}

// NewRedisEventStream creates a new Redis event stream
func NewRedisEventStream(cfg *RedisConfig, logger *logrus.Logger) *RedisEventStream {
	ctx, cancel := context.WithCancel(context.Background())

	if cfg == nil {
		cfg = &RedisConfig{
			Host:            "localhost",
			Port:            6379,
			Password:        "",
			DB:              0,
			EnableStreaming: true,
			Prefix:          "axiom",
			StreamName:      "service-events",
			ConsumerGroup:   "axiom-consumers",
			ConsumerName:    "consumer-1",
			MaxStreamLength: 10000,
			PollInterval:    1 * time.Second,
		}
	}

	return &RedisEventStream{
		config:      cfg,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		subscribers: make(map[string][]chan *Event),
		streamName:  getStreamName(cfg.Prefix, cfg.StreamName),
		initialized: false,
	}
}

// getStreamName returns the full stream name with prefix
func getStreamName(prefix, streamName string) string {
	if prefix != "" {
		return fmt.Sprintf("%s:%s", prefix, streamName)
	}
	return streamName
}

// initialize creates the Redis client connection
func (s *RedisEventStream) initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized && s.client != nil {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: s.config.Password,
		DB:       s.config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	if err := s.client.Ping(ctx).Err(); err != nil {
		s.logger.WithError(err).Error("Failed to connect to Redis")
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create stream with initial length
	if s.config.EnableStreaming {
		s.createStreamWithLength(ctx, s.streamName, s.config.MaxStreamLength)
	}

	s.initialized = true
	s.logger.WithFields(logrus.Fields{
		"addr": addr,
		"db":   s.config.DB,
		"stream": s.streamName,
	}).Info("Redis event stream initialized")

	return nil
}

// createStreamWithLength creates or configures the stream with a maximum length
func (s *RedisEventStream) createStreamWithLength(ctx context.Context, streamName string, maxLength int64) {
	// Use XINFO to check if stream exists, if not create it
	// Redis automatically creates streams when we add to them
	// Trim is done on insert using the XADD command
}

// AddEvent adds an event to the Redis stream
func (s *RedisEventStream) AddEvent(event *Event) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return ErrStreamNotConnected
	}

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	// Convert event to JSON
	data, err := s.eventToMap(event)
	if err != nil {
		return err
	}

	// Add event to stream with automatic trimming
	args := redis.Args{streamName, "MAXLEN", "~", s.config.MaxStreamLength}
	for k, v := range data {
		args = args.Add(k).Add(v)
	}

	if _, err := client.XAdd(ctx, redis.XAddArgs{
		Stream: streamName,
		Values: args[2:], // Skip stream name and MAXLEN clause
	}).Result(); err != nil {
		return fmt.Errorf("failed to add event to stream: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"event_type": event.Type,
		"stream":     streamName,
	}).Debug("Event added to stream")

	return nil
}

// eventToMap converts an Event to a map for Redis storage
func (s *RedisEventStream) eventToMap(event *Event) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Add event type
	result["type"] = eventTypeToString(event.Type)

	// Add timestamp
	result["timestamp"] = event.Timestamp.Format(time.RFC3339)

	// Add error if present
	if event.Error != nil {
		result["error"] = event.Error.Error()
	}

	// Add container data if present
	if event.Container != nil {
		containerJSON, err := json.Marshal(event.Container)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal container: %w", err)
		}
		result["container"] = string(containerJSON)
	}

	// Add image data if present
	if event.Image != nil {
		imageJSON, err := json.Marshal(event.Image)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal image: %w", err)
		}
		result["image"] = string(imageJSON)
	}

	// Add network data if present
	if event.Network != nil {
		networkJSON, err := json.Marshal(event.Network)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal network: %w", err)
		}
		result["network"] = string(networkJSON)
	}

	// Add pod data if present
	if event.Pod != nil {
		podJSON, err := json.Marshal(event.Pod)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal pod: %w", err)
		}
		result["pod"] = string(podJSON)
	}

	// Add deployment data if present
	if event.Deployment != nil {
		deploymentJSON, err := json.Marshal(event.Deployment)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal deployment: %w", err)
		}
		result["deployment"] = string(deploymentJSON)
	}

	// Add service data if present
	if event.Service != nil {
		serviceJSON, err := json.Marshal(event.Service)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal service: %w", err)
		}
		result["service"] = string(serviceJSON)
	}

	// Add metadata
	for k, v := range event.Metadata {
		result[k] = v
	}

	return result, nil
}

// Subscribe subscribes to a channel for specific event types
func (s *RedisEventStream) Subscribe(eventType EventType, ch chan *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := eventTypeToString(eventType)
	s.subscribers[key] = append(s.subscribers[key], ch)

	s.logger.WithFields(logrus.Fields{
		"event_type": key,
		"subscriber": len(s.subscribers[key]),
	}).Debug("Event stream subscriber added")

	return nil
}

// Unsubscribe unsubscribes from a channel
func (s *RedisEventStream) Unsubscribe(eventType EventType, ch chan *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := eventTypeToString(eventType)
	subscribers := s.subscribers[key]

	for i, sub := range subscribers {
		if sub == ch {
			s.subscribers[key] = append(subscribers[:i], subscribers[i+1:]...)
			close(ch)
			s.logger.WithFields(logrus.Fields{
				"event_type": key,
				"subscriber": len(s.subscribers[key]),
			}).Debug("Event stream subscriber removed")
			return nil
		}
	}

	return fmt.Errorf("subscriber not found for event type: %s", key)
}

// StartConsumer starts consuming events from the Redis stream
func (s *RedisEventStream) StartConsumer(consumerName string, handler func(*Event)) error {
	if handler == nil {
		return ErrNilHandler
	}

	return s.StartConsumerGroup(consumerName, s.config.ConsumerGroup, handler)
}

// StartConsumerGroup starts consuming events from a Redis consumer group
func (s *RedisEventStream) StartConsumerGroup(consumerName, groupName string, handler func(*Event)) error {
	if handler == nil {
		return ErrNilHandler
	}

	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return ErrStreamNotConnected
	}

	go func() {
		consumeCtx, cancel := context.WithCancel(s.ctx)
		defer cancel()

		// Create consumer group if it doesn't exist
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		err := client.XGroupCreateMkStream(ctx, s.streamName, groupName, "0").Err()
		cancel()

		if err != nil && err != redis.ErrBusy {
			s.logger.WithFields(logrus.Fields{
				"stream":   s.streamName,
				"group":    groupName,
				"consumer": consumerName,
				"error":    err,
			}).Error("Failed to create consumer group")
			return
		}

		s.logger.WithFields(logrus.Fields{
			"stream":   s.streamName,
			"group":    groupName,
			"consumer": consumerName,
		}).Info("Starting event consumer")

		for {
			select {
			case <-consumeCtx.Done():
				return
			default:
				s.consumeFromStream(consumerName, groupName, handler)
				time.Sleep(s.config.PollInterval)
			}
		}
	}()

	return nil
}

// consumeFromStream consumes messages from the Redis stream
func (s *RedisEventStream) consumeFromStream(consumerName, groupName string, handler func(*Event)) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	// Read messages from consumer group
	msgs, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Streams:  []string{s.streamName, ">"},
		Group:    groupName,
		Consumer: consumerName,
		Count:    10,
		Block:    1000,
	}).Result()

	if err != nil && err != redis.ErrClosed {
		s.logger.WithFields(logrus.Fields{
			"stream": s.streamName,
			"error":  err,
		}).Warn("Failed to read from stream")
		return
	}

	if len(msgs) == 0 {
		return
	}

	// Process messages
	for stream, messages := range msgs {
		s.logger.WithField("stream", stream).Debug("Received messages from stream")

		for _, msg := range messages {
			s.processMessage(msg, handler)
		}
	}
}

// processMessage processes a message from the stream
func (s *RedisEventStream) processMessage(msg redis.XMessage, handler func(*Event)) {
	event := &Event{
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Parse metadata
	for k, v := range msg.Values {
		switch vv := v.(type) {
		case string:
			event.Metadata[k] = vv
		case int64:
			event.Metadata[k] = vv
		case float64:
			event.Metadata[k] = vv
		default:
			event.Metadata[k] = vv
		}

		// Convert timestamp string to time
		if k == "timestamp" {
			if t, err := time.Parse(time.RFC3339, vv); err == nil {
				event.Timestamp = t
			}
		}
	}

	// Parse container data
	if containerJSON, ok := event.Metadata["container"].(string); ok {
		var container Container
		if err := json.Unmarshal([]byte(containerJSON), &container); err == nil {
			event.Container = &container
			delete(event.Metadata, "container")
		}
	}

	// Parse image data
	if imageJSON, ok := event.Metadata["image"].(string); ok {
		var image Image
		if err := json.Unmarshal([]byte(imageJSON), &image); err == nil {
			event.Image = &image
			delete(event.Metadata, "image")
		}
	}

	// Parse network data
	if networkJSON, ok := event.Metadata["network"].(string); ok {
		var network Network
		if err := json.Unmarshal([]byte(networkJSON), &network); err == nil {
			event.Network = &network
			delete(event.Metadata, "network")
		}
	}

	// Parse pod data
	if podJSON, ok := event.Metadata["pod"].(string); ok {
		var pod PodDetail
		if err := json.Unmarshal([]byte(podJSON), &pod); err == nil {
			event.Pod = &pod
			delete(event.Metadata, "pod")
		}
	}

	// Parse deployment data
	if deploymentJSON, ok := event.Metadata["deployment"].(string); ok {
		var deployment ServiceResource
		if err := json.Unmarshal([]byte(deploymentJSON), &deployment); err == nil {
			event.Deployment = &deployment
			delete(event.Metadata, "deployment")
		}
	}

	// Parse service data
	if serviceJSON, ok := event.Metadata["service"].(string); ok {
		var service ServiceResource
		if err := json.Unmarshal([]byte(serviceJSON), &service); err == nil {
			event.Service = &service
			delete(event.Metadata, "service")
		}
	}

	// Call handler
	if handler != nil {
		handler(event)
	}

	// Acknowledge message
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client != nil {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()

		client.XAck(ctx, s.streamName, s.config.ConsumerGroup, msg.ID)
	}
}

// PublishEvent publishes an event to the stream
func (s *RedisEventStream) PublishEvent(event *Event) error {
	return s.AddEvent(event)
}

// StreamLength returns the number of entries in the stream
func (s *RedisEventStream) StreamLength() (int64, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return 0, ErrStreamNotConnected
	}

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	return client.XLen(ctx, s.streamName).Result()
}

// GetMessages retrieves recent messages from the stream
func (s *RedisEventStream) GetMessages(count int64) ([]*Event, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, ErrStreamNotConnected
	}

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	result, err := client.XRevRangeN(ctx, s.streamName, -1, count).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read messages from stream: %w", err)
	}

	events := make([]*Event, 0, len(result))
	for _, msg := range result {
		event := s.messageToEvent(msg)
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// messageToEvent converts a Redis message to an Event
func (s *RedisEventStream) messageToEvent(msg redis.XMessage) *Event {
	event := &Event{
		ID:        msg.ID,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Parse metadata
	for k, v := range msg.Values {
		switch vv := v.(type) {
		case string:
			event.Metadata[k] = vv
		case int64:
			event.Metadata[k] = vv
		case float64:
			event.Metadata[k] = vv
		default:
			event.Metadata[k] = vv
		}

		// Convert timestamp string to time
		if k == "timestamp" {
			if t, err := time.Parse(time.RFC3339, vv); err == nil {
				event.Timestamp = t
			}
		}
	}

	// Parse container data
	if containerJSON, ok := event.Metadata["container"].(string); ok {
		var container Container
		if err := json.Unmarshal([]byte(containerJSON), &container); err == nil {
			event.Container = &container
		}
	}

	// Parse image data
	if imageJSON, ok := event.Metadata["image"].(string); ok {
		var image Image
		if err := json.Unmarshal([]byte(imageJSON), &image); err == nil {
			event.Image = &image
		}
	}

	// Parse network data
	if networkJSON, ok := event.Metadata["network"].(string); ok {
		var network Network
		if err := json.Unmarshal([]byte(networkJSON), &network); err == nil {
			event.Network = &network
		}
	}

	// Parse pod data
	if podJSON, ok := event.Metadata["pod"].(string); ok {
		var pod PodDetail
		if err := json.Unmarshal([]byte(podJSON), &pod); err == nil {
			event.Pod = &pod
		}
	}

	// Parse deployment data
	if deploymentJSON, ok := event.Metadata["deployment"].(string); ok {
		var deployment ServiceResource
		if err := json.Unmarshal([]byte(deploymentJSON), &deployment); err == nil {
			event.Deployment = &deployment
		}
	}

	// Parse service data
	if serviceJSON, ok := event.Metadata["service"].(string); ok {
		var service ServiceResource
		if err := json.Unmarshal([]byte(serviceJSON), &service); err == nil {
			event.Service = &service
		}
	}

	return event
}

// StopConsumer stops consuming from the stream
func (s *RedisEventStream) StopConsumer(consumerName string) {
	s.logger.WithField("consumer", consumerName).Info("Stopping consumer")
	// Consumers are stopped via context cancellation
}

// Shutdown stops the Redis event stream
func (s *RedisEventStream) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	if s.client != nil {
		s.client.Close()
	}

	// Close all subscriber channels
	for _, channels := range s.subscribers {
		for _, ch := range channels {
			close(ch)
		}
	}

	s.initialized = false

	s.logger.Info("Redis event stream shut down")
}

// GetSubscriberCount returns the number of subscribers for an event type
func (s *RedisEventStream) GetSubscriberCount(eventType EventType) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := eventTypeToString(eventType)
	return len(s.subscribers[key])
}

// BroadcastEvent broadcasts an event to all subscribers
func (s *RedisEventStream) BroadcastEvent(event *Event) {
	s.mu.RLock()
	eventType := event.Type
	subscribers := make([]chan *Event, 0)
	s.mu.RUnlock()

	// Find matching subscribers
	s.mu.RLock()
	for key, channels := range s.subscribers {
		if eventTypeToString(eventType) == key {
			for _, ch := range channels {
				select {
				case ch <- event:
				default:
					s.logger.WithField("subscriber", ch).Debug("Subscriber channel full")
				}
			}
		}
	}
	s.mu.RUnlock()

	// Also broadcast to general subscribers
	s.mu.RLock()
	if generalCh, ok := s.subscribers["*"]; ok {
		for _, ch := range generalCh {
			select {
			case ch <- event:
			default:
				s.logger.WithField("subscriber", ch).Debug("Subscriber channel full")
			}
		}
	}
	s.mu.RUnlock()

	// Also broadcast to all subscribers
	s.mu.RLock()
	for _, channels := range s.subscribers {
		for _, ch := range channels {
			select {
			case ch <- event:
			default:
				s.logger.WithField("subscriber", ch).Debug("Subscriber channel full")
			}
		}
	}
	s.mu.RUnlock()
}

// IsConnected returns true if the Redis stream is connected
func (s *RedisEventStream) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client != nil && s.initialized
}

// GetClient returns the Redis client
func (s *RedisEventStream) GetClient() *redis.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}
