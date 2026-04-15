package gitlab

import "errors"

// Errors for GitLab CI integration
var (
	// ErrMissingAPIURL is returned when GitLab API URL is missing
	ErrMissingAPIURL = errors.New("GitLab API URL is required")

	// ErrMissingAPIToken is returned when API token is missing
	ErrMissingAPIToken = errors.New("GitLab API token is required")

	// ErrInvalidTimeout is returned when timeout is invalid
	ErrInvalidTimeout = errors.New("timeout must be at least 1 second")

	// ErrInsecureWebhook is returned when webhook is configured insecurely
	ErrInsecureWebhook = errors.New("webhook secret should not be used without SSL verification")

	// ErrInvalidPort is returned when port is out of valid range
	ErrInvalidPort = errors.New("port must be between 1 and 65535")

	// ErrInvalidProjectID is returned when project ID is invalid
	ErrInvalidProjectID = errors.New("invalid project ID")

	// ErrPipelineNotFound is returned when pipeline is not found
	ErrPipelineNotFound = errors.New("pipeline not found")

	// ErrJobNotFound is returned when job is not found
	ErrJobNotFound = errors.New("job not found")

	// ErrRunnerNotFound is returned when runner is not found
	ErrRunnerNotFound = errors.New("runner not found")

	// ErrMRNotFound is returned when merge request is not found
	ErrMRNotFound = errors.New("merge request not found")

	// ErrInvalidMRState is returned when merge request state is invalid for action
	ErrInvalidMRState = errors.New("merge request state does not allow this action")

	// ErrRateLimited is returned when API rate limit is exceeded
	ErrRateLimited = errors.New("GitLab API rate limit exceeded")

	// ErrUnreachable is returned when GitLab instance is unreachable
	ErrUnreachable = errors.New("GitLab instance is unreachable")

	// ErrAuthenticationFailed is returned when authentication fails
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrWebhookSignatureInvalid is returned when webhook signature is invalid
	ErrWebhookSignatureInvalid = errors.New("webhook signature is invalid")

	// ErrWebhookEventNotAllowed is returned when webhook event type is not allowed
	ErrWebhookEventNotAllowed = errors.New("webhook event type is not allowed")
)

// Error is the base error type for GitLab CI integration
type Error struct {
	Code    string
	Message string
	Err     error
}

// Error implements error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new GitLab CI error
func NewError(code, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsError returns true if the error is a GitLab CI error with the given code
func IsError(err error, code string) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == code
	}
	return false
}
