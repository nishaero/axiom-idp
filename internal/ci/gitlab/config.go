package gitlab

import (
	"time"
)

// Config contains GitLab CI integration configuration
type Config struct {
	// Client configuration
	Client ClientConfig

	// Webhook configuration
	Webhook WebhookConfig

	// Orchestration configuration
	Orchestration OrchestrationConfig

	// API configuration
	API APIConfig

	// Feature flags
	Features FeatureFlags
}

// APIConfig contains API configuration
type APIConfig struct {
	// Port is the API server port
	Port int `json:"port" yaml:"port"`

	// Host is the API server host
	Host string `json:"host" yaml:"host"`

	// EnableCORS enables CORS headers
	EnableCORS bool `json:"enable_cors" yaml:"enable_cors"`

	// CORSOrigins is the list of allowed origins
	CORSOrigins []string `json:"cors_origins" yaml:"cors_origins"`

	// EnableAuth enables authentication for API endpoints
	EnableAuth bool `json:"enable_auth" yaml:"enable_auth"`

	// AuthMethods lists supported authentication methods
	AuthMethods []string `json:"auth_methods" yaml:"auth_methods"`
}

// FeatureFlags contains feature flag configuration
type FeatureFlags struct {
	// EnablePipelineTriggering enables pipeline triggering from webhooks
	EnablePipelineTriggering bool `json:"enable_pipeline_triggering" yaml:"enable_pipeline_triggering"`

	// EnableJobCancellation enables job cancellation
	EnableJobCancellation bool `json:"enable_job_cancellation" yaml:"enable_job_cancellation"`

	// EnableJobRetry enables job retry
	EnableJobRetry bool `json:"enable_job_retry" yaml:"enable_job_retry"`

	// EnableMergeRequestIntegration enables MR integration
	EnableMergeRequestIntegration bool `json:"enable_merge_request_integration" yaml:"enable_merge_request_integration"`

	// EnableCostReporting enables cost reporting
	EnableCostReporting bool `json:"enable_cost_reporting" yaml:"enable_cost_reporting"`

	// EnableServiceCatalog enables service catalog updates
	EnableServiceCatalog bool `json:"enable_service_catalog" yaml:"enable_service_catalog"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Client: ClientConfig{
			APIURL:     "https://gitlab.com",
			APIToken:   "placeholder-token",
			Timeout:    30 * time.Second,
			RetryCount: 3,
			RetryDelay: 5 * time.Second,
		},
		Webhook: WebhookConfig{
			Path:           "/api/v1/ci/gitlab/webhook",
			VerifySSL:      true,
			EnableAuditLog: true,
		},
		Orchestration: OrchestrationConfig{
			MetricsCollectionInterval: 5 * time.Minute,
			MaxConcurrentPipelines:    10,
			CostPerMinute:             0.002,
		},
		API: APIConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			EnableCORS:  true,
			CORSOrigins: []string{"*"},
		},
		Features: FeatureFlags{
			EnablePipelineTriggering:      true,
			EnableJobCancellation:         true,
			EnableJobRetry:                true,
			EnableMergeRequestIntegration: true,
			EnableCostReporting:           true,
			EnableServiceCatalog:          true,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate client config
	if c.Client.APIURL == "" {
		return ErrMissingAPIURL
	}

	if c.Client.APIToken == "" {
		return ErrMissingAPIToken
	}

	if c.Client.Timeout < time.Second {
		return ErrInvalidTimeout
	}

	// Validate webhook config
	if c.Webhook.Secret != "" && !c.Webhook.VerifySSL {
		return ErrInsecureWebhook
	}

	// Validate API config
	if c.API.Port < 1 || c.API.Port > 65535 {
		return ErrInvalidPort
	}

	return nil
}

// Merge merges two configurations
func (c *Config) Merge(other *Config) *Config {
	result := *c

	if other.Client.APIURL != "" {
		result.Client.APIURL = other.Client.APIURL
	}
	if other.Client.APIToken != "" {
		result.Client.APIToken = other.Client.APIToken
	}
	if other.Client.Timeout > 0 {
		result.Client.Timeout = other.Client.Timeout
	}
	if other.Client.RetryCount > 0 {
		result.Client.RetryCount = other.Client.RetryCount
	}
	if other.Client.RetryDelay > 0 {
		result.Client.RetryDelay = other.Client.RetryDelay
	}

	if other.Webhook.Path != "" {
		result.Webhook.Path = other.Webhook.Path
	}
	if other.Webhook.Secret != "" {
		result.Webhook.Secret = other.Webhook.Secret
	}
	if other.Webhook.VerifySSL {
		result.Webhook.VerifySSL = other.Webhook.VerifySSL
	}

	if other.Orchestration.MetricsCollectionInterval > 0 {
		result.Orchestration.MetricsCollectionInterval = other.Orchestration.MetricsCollectionInterval
	}
	if other.Orchestration.MaxConcurrentPipelines > 0 {
		result.Orchestration.MaxConcurrentPipelines = other.Orchestration.MaxConcurrentPipelines
	}

	if other.API.Port > 0 {
		result.API.Port = other.API.Port
	}
	if other.API.Host != "" {
		result.API.Host = other.API.Host
	}

	return &result
}
