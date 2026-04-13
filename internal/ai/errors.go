package ai

import (
	"errors"
	"fmt"
)

// AIError represents an AI service error
type AIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	Err       error  `json:"-"`
	Timestamp string `json:"timestamp"`
}

// Error implements error interface for AIError
func (e *AIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AIError) Unwrap() error {
	return e.Err
}

// Error codes for AI service errors
var (
	// ErrInvalidQuery indicates an invalid query format
	ErrInvalidQuery = errors.New("invalid query format")

	// ErrQueryEmpty indicates an empty query
	ErrQueryEmpty = errors.New("query cannot be empty")

	// ErrEmbeddingFailed indicates embedding generation failed
	ErrEmbeddingFailed = errors.New("embedding generation failed")

	// ErrVectorSearchFailed indicates vector search failed
	ErrVectorSearchFailed = errors.New("vector search failed")

	// ErrUserNotFound indicates user context not found
	ErrUserNotFound = errors.New("user context not found")

	// ErrServiceNotFound indicates service not found
	ErrServiceNotFound = errors.New("service not found")

	// ErrRecommendationFailed indicates recommendation generation failed
	ErrRecommendationFailed = errors.New("recommendation generation failed")

	// ErrPromptEngineFailed indicates prompt generation failed
	ErrPromptEngineFailed = errors.New("prompt generation failed")

	// ErrLLMUnreachable indicates LLM service is unreachable
	ErrLLMUnreachable = errors.New("LLM service is unreachable")

	// ErrRateLimited indicates rate limit exceeded
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrValidationError indicates input validation failed
	ErrValidationError = errors.New("input validation failed")

	// ErrContextExceeded indicates context size exceeded limits
	ErrContextExceeded = errors.New("context size exceeds maximum allowed")

	// ErrInvalidEmbedding indicates invalid embedding vector
	ErrInvalidEmbedding = errors.New("invalid embedding vector dimensions")
)

// IsAIError checks if an error is an AIError
func IsAIError(err error) bool {
	_, ok := err.(*AIError)
	return ok
}

// GetAIError extracts AIError from error chain
func GetAIError(err error) *AIError {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr
	}
	return nil
}

// WrapError wraps an error with AI error context
func WrapError(code string, err error, details string) *AIError {
	return &AIError{
		Code:    code,
		Message: formatErrorCode(code),
		Details: details,
		Err:     err,
		Timestamp: formatTimestamp(),
	}
}

// NewAIError creates a new AIError
func NewAIError(code, message, details string) *AIError {
	return &AIError{
		Code:    code,
		Message: message,
		Details: details,
		Timestamp: formatTimestamp(),
	}
}

// formatErrorCode formats error code to readable message
func formatErrorCode(code string) string {
	messages := map[string]string{
		"INVALID_QUERY":          "The query format is invalid",
		"QUERY_EMPTY":            "The query cannot be empty",
		"EMBEDDING_FAILED":       "Failed to generate embedding vector",
		"VECTOR_SEARCH_FAILED":   "Failed to perform vector search",
		"USER_NOT_FOUND":         "User context not found",
		"SERVICE_NOT_FOUND":      "Service not found in catalog",
		"RECOMMENDATION_FAILED":  "Failed to generate recommendations",
		"PROMPT_ENGINE_FAILED":   "Failed to process prompt",
		"LLM_UNREACHABLE":        "LLM service is currently unavailable",
		"RATE_LIMITED":           "Rate limit exceeded. Please try again later.",
		"VALIDATION_FAILED":      "Input validation failed",
		"CONTEXT_EXCEEDED":       "Context size exceeds maximum allowed",
		"INVALID_EMBEDDING":      "Invalid embedding vector dimensions",
	}

	if msg, ok := messages[code]; ok {
		return msg
	}

	return code
}

// formatTimestamp formats current timestamp
func formatTimestamp() string {
	return "2006-01-02T15:04:05.000Z07:00"
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements error interface for ValidationErrors
func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}

	var sb []string
	for _, err := range e.Errors {
		sb = append(sb, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}

	return fmt.Sprintf("validation failed: %s", sb[0])
}

// AddError adds a validation error
func (e *ValidationErrors) AddError(field, code, message string) {
	e.Errors = append(e.Errors, ValidationError{
		Field:   field,
		Code:    code,
		Message: message,
	})
}

// HasErrors checks if there are any validation errors
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// NewValidationErrors creates new validation errors
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]ValidationError, 0),
	}
}

// QueryValidationErrors validates query parameters
func QueryValidationErrors(query *QueryContext) *ValidationErrors {
	errors := NewValidationErrors()

	if query == nil {
		errors.AddError("query", "VALIDATION_FAILED", "Query context is nil")
		return errors
	}

	if query.Query == "" {
		errors.AddError("query.text", "QUERY_EMPTY", "Query text cannot be empty")
	}

	if query.ContextSize < 0 {
		errors.AddError("query.context_size", "VALIDATION_FAILED", "Context size must be non-negative")
	}

	if query.ContextSize > 100000 {
		errors.AddError("query.context_size", "CONTEXT_EXCEEDED", "Context size exceeds maximum allowed")
	}

	if query.UserID == "" {
		errors.AddError("query.user_id", "VALIDATION_FAILED", "User ID is recommended for context tracking")
	}

	return errors
}

// RecommendationValidationErrors validates recommendation parameters
func RecommendationValidationErrors(userID string, query string, limit int) *ValidationErrors {
	errors := NewValidationErrors()

	if userID == "" {
		errors.AddError("user_id", "VALIDATION_FAILED", "User ID is required")
	}

	if query == "" {
		errors.AddError("query", "QUERY_EMPTY", "Query cannot be empty")
	}

	if limit <= 0 {
		errors.AddError("limit", "VALIDATION_FAILED", "Limit must be positive")
	} else if limit > 100 {
		errors.AddError("limit", "CONTEXT_EXCEEDED", "Limit exceeds maximum allowed (100)")
	}

	return errors
}

// EmbeddingValidationErrors validates embedding parameters
func EmbeddingValidationErrors(serviceID, serviceName string, vector []float32) *ValidationErrors {
	errors := NewValidationErrors()

	if serviceID == "" {
		errors.AddError("service_id", "VALIDATION_FAILED", "Service ID is required")
	}

	if serviceName == "" {
		errors.AddError("service_name", "VALIDATION_FAILED", "Service name is required")
	}

	if len(vector) == 0 {
		errors.AddError("vector", "INVALID_EMBEDDING", "Embedding vector cannot be empty")
	} else if len(vector) < 128 || len(vector) > 4096 {
		errors.AddError("vector", "INVALID_EMBEDDING", fmt.Sprintf("Embedding vector dimension must be between 128 and 4096, got %d", len(vector)))
	}

	return errors
}
