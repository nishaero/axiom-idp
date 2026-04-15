package catalog

import "errors"

var (
	// Core errors
	ErrServiceNotFound      = errors.New("service not found in catalog")
	ErrServiceAlreadyExists = errors.New("service already exists in catalog")

	// Docker errors
	ErrDockerNotInitialized = errors.New("Docker client not initialized")
	ErrDockerNotConnected   = errors.New("Docker client not connected")

	// Kubernetes errors
	ErrK8sNotInitialized = errors.New("Kubernetes client not initialized")
	ErrK8sNotConnected   = errors.New("Kubernetes client not connected")

	// Redis errors
	ErrStreamNotConnected = errors.New("Redis event stream not connected")
	ErrStreamNotFound     = errors.New("Redis stream not found")

	// Redis stream errors
	ErrNilHandler = errors.New("handler function is nil")

	// WebSocket errors
	ErrWebSocketNotInitialized = errors.New("WebSocket hub not initialized")
	ErrWebSocketClosed         = errors.New("WebSocket connection closed")
	ErrWebSocketFull           = errors.New("WebSocket connection buffer full")

	// Event errors
	ErrEventAlreadyPublished = errors.New("event already published")
	ErrEventParseFailed      = errors.New("failed to parse event")

	// Cache errors
	ErrCacheFull       = errors.New("cache is full")
	ErrCacheNotFound   = errors.New("cache entry not found")
	ErrCacheEviction   = errors.New("cache entry evicted")

	// Health check errors
	ErrHealthCheckFailed = errors.New("health check failed")
	ErrHealthTimeout     = errors.New("health check timeout")
	ErrHealthUnhealthy   = errors.New("health check unhealthy")
)

// Custom error types for better error handling

// DockerError wraps Docker-related errors
type DockerError struct {
	Operation string
	Container string
	Message   string
	Err       error
}

func (e *DockerError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *DockerError) Unwrap() error {
	return e.Err
}

// K8sError wraps Kubernetes-related errors
type K8sError struct {
	Operation string
	Namespace string
	Resource  string
	Message   string
	Err       error
}

func (e *K8sError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *K8sError) Unwrap() error {
	return e.Err
}

// RedisError wraps Redis-related errors
type RedisError struct {
	Operation string
	Stream    string
	Message   string
	Err       error
}

func (e *RedisError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *RedisError) Unwrap() error {
	return e.Err
}

// WebSocketError wraps WebSocket-related errors
type WebSocketError struct {
	Operation string
	Conn      string
	Message   string
	Err       error
}

func (e *WebSocketError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *WebSocketError) Unwrap() error {
	return e.Err
}

// EventError wraps event-related errors
type EventError struct {
	EventID string
	Message string
	Err     error
}

func (e *EventError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *EventError) Unwrap() error {
	return e.Err
}

// HealthError wraps health check errors
type HealthError struct {
	Name        string
	Expected    string
	Actual      string
	Duration    string
	Err         error
}

func (e *HealthError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "Health check failed: expected " + e.Expected + " but got " + e.Actual
}

func (e *HealthError) Unwrap() error {
	return e.Err
}

// IsDockerError checks if an error is a Docker error
func IsDockerError(err error) bool {
	_, ok := err.(*DockerError)
	return ok
}

// IsK8sError checks if an error is a Kubernetes error
func IsK8sError(err error) bool {
	_, ok := err.(*K8sError)
	return ok
}

// IsRedisError checks if an error is a Redis error
func IsRedisError(err error) bool {
	_, ok := err.(*RedisError)
	return ok
}

// IsWebSocketError checks if an error is a WebSocket error
func IsWebSocketError(err error) bool {
	_, ok := err.(*WebSocketError)
	return ok
}

// IsHealthError checks if an error is a health check error
func IsHealthError(err error) bool {
	_, ok := err.(*HealthError)
	return ok
}

// IsEventError checks if an error is an event error
func IsEventError(err error) bool {
	_, ok := err.(*EventError)
	return ok
}

// NewDockerError creates a new Docker error
func NewDockerError(operation, container, message string, err error) *DockerError {
	return &DockerError{
		Operation: operation,
		Container: container,
		Message:   message,
		Err:       err,
	}
}

// NewK8sError creates a new Kubernetes error
func NewK8sError(operation, namespace, resource, message string, err error) *K8sError {
	return &K8sError{
		Operation: operation,
		Namespace: namespace,
		Resource:  resource,
		Message:   message,
		Err:       err,
	}
}

// NewRedisError creates a new Redis error
func NewRedisError(operation, stream, message string, err error) *RedisError {
	return &RedisError{
		Operation: operation,
		Stream:    stream,
		Message:   message,
		Err:       err,
	}
}

// NewWebSocketError creates a new WebSocket error
func NewWebSocketError(operation, conn, message string, err error) *WebSocketError {
	return &WebSocketError{
		Operation: operation,
		Conn:      conn,
		Message:   message,
		Err:       err,
	}
}

// NewEventError creates a new event error
func NewEventError(eventID, message string, err error) *EventError {
	return &EventError{
		EventID: eventID,
		Message: message,
		Err:     err,
	}
}

// NewHealthError creates a new health check error
func NewHealthError(name, expected, actual string, duration string, err error) *HealthError {
	return &HealthError{
		Name:     name,
		Expected: expected,
		Actual:   actual,
		Duration: duration,
		Err:      err,
	}
}
