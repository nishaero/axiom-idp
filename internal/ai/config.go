package ai

import (
	"fmt"
	"time"
)

// AIConfig holds the AI service configuration
type AIConfig struct {
	// OpenAI configuration
	OpenAI APIConfig `yaml:"openai" json:"openai"`

	// PGVector configuration
	PGVector PGVectorConfig `yaml:"pgvector" json:"pgvector"`

	// Engine settings
	Engine EngineConfig `yaml:"engine" json:"engine"`

	// Context settings
	Context ContextConfig `yaml:"context" json:"context"`
}

// APIConfig contains API configuration
type APIConfig struct {
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	APIKey          string        `yaml:"api_key" json:"-"`
	BaseURL         string        `yaml:"base_url" json:"base_url"`
	CompletionModel string        `yaml:"completion_model" json:"completion_model"`
	EmbeddingModel  string        `yaml:"embedding_model" json:"embedding_model"`
	Timeout         time.Duration `yaml:"timeout" json:"timeout"`
	RateLimit       int           `yaml:"rate_limit" json:"rate_limit"`
	RateLimitWindow time.Duration `yaml:"rate_limit_window" json:"rate_limit_window"`
	Temperature     float64       `yaml:"temperature" json:"temperature"`
	MaxTokens       int           `yaml:"max_tokens" json:"max_tokens"`
	EnableCache     bool          `yaml:"enable_cache" json:"enable_cache"`
}

// PGVectorConfig contains PostgreSQL pgvector configuration
type PGVectorConfig struct {
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	DSN             string        `yaml:"dsn" json:"-"`
	CollectionSize  int           `yaml:"collection_size" json:"collection_size"`
	COSIMDistance   bool          `yaml:"cosim_distance" json:"cosim_distance"`
	TopK            int           `yaml:"top_k" json:"top_k"`
	Threshold       float32       `yaml:"threshold" json:"threshold"`
	MaxConnections  int           `yaml:"max_connections" json:"max_connections"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout" json:"connection_timeout"`
}

// EngineConfig contains engine settings
type EngineConfig struct {
	MaxUserContexts       int           `yaml:"max_user_contexts" json:"max_user_contexts"`
	DefaultContextSize    int           `yaml:"default_context_size" json:"default_context_size"`
	MaxContextSize        int           `yaml:"max_context_size" json:"max_context_size"`
	RecommendationLimit   int           `yaml:"recommendation_limit" json:"recommendation_limit"`
	SearchTopK            int           `yaml:"search_top_k" json:"search_top_k"`
	EnableMetrics         bool          `yaml:"enable_metrics" json:"enable_metrics"`
	EnableLogging         bool          `yaml:"enable_logging" json:"enable_logging"`
	EmbeddingDimension    int           `yaml:"embedding_dimension" json:"embedding_dimension"`
	SeedRandom            bool          `yaml:"seed_random" json:"seed_random"`
}

// ContextConfig contains context window settings
type ContextConfig struct {
	DefaultSize           int           `yaml:"default_size" json:"default_size"`
	MaxSize               int           `yaml:"max_size" json:"max_size"`
	OptimizationEnabled   bool          `yaml:"optimization_enabled" json:"optimization_enabled"`
	CleanupInterval       time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
	UserContextTTL        time.Duration `yaml:"user_context_ttl" json:"user_context_ttl"`
	QueryRetention        int           `yaml:"query_retention" json:"query_retention"`
}

// DefaultAIConfig returns default AI configuration
func DefaultAIConfig() *AIConfig {
	return &AIConfig{
		OpenAI: APIConfig{
			Enabled:         false,
			APIKey:          "",
			BaseURL:         "https://api.openai.com/v1",
			CompletionModel: "gpt-3.5-turbo",
			EmbeddingModel:  "text-embedding-ada-002",
			Timeout:         30 * time.Second,
			RateLimit:       100,
			RateLimitWindow: time.Minute,
			Temperature:     0.7,
			MaxTokens:       2048,
			EnableCache:     true,
		},
		PGVector: PGVectorConfig{
			Enabled:          false,
			DSN:              "",
			CollectionSize:   100,
			COSIMDistance:    true,
			TopK:             10,
			Threshold:        0.7,
			MaxConnections:   10,
			ConnectionTimeout: 5 * time.Second,
		},
		Engine: EngineConfig{
			MaxUserContexts:    10000,
			DefaultContextSize: 4000,
			MaxContextSize:     32000,
			RecommendationLimit: 10,
			SearchTopK:         50,
			EnableMetrics:      true,
			EnableLogging:      true,
			EmbeddingDimension: 1536,
			SeedRandom:         true,
		},
		Context: ContextConfig{
			DefaultSize:           4000,
			MaxSize:               32000,
			OptimizationEnabled:   true,
			CleanupInterval:       1 * time.Hour,
			UserContextTTL:        24 * time.Hour,
			QueryRetention:        10,
		},
	}
}

// Validate validates the AI configuration
func (c *AIConfig) Validate() error {
	if c.OpenAI.RateLimit < 0 {
		return fmt.Errorf("invalid rate_limit: %d", c.OpenAI.RateLimit)
	}

	if c.OpenAI.Timeout < 1*time.Second {
		return fmt.Errorf("invalid timeout: must be at least 1 second")
	}

	if c.OpenAI.Temperature < 0 || c.OpenAI.Temperature > 2 {
		return fmt.Errorf("invalid temperature: must be between 0 and 2")
	}

	if c.OpenAI.MaxTokens < 100 {
		return fmt.Errorf("invalid max_tokens: must be at least 100")
	}

	if c.PGVector.MaxConnections < 1 {
		return fmt.Errorf("invalid max_connections: must be at least 1")
	}

	if c.Engine.MaxContextSize < c.Engine.DefaultContextSize {
		return fmt.Errorf("max_context_size must be >= default_context_size")
	}

	if c.Context.QueryRetention < 0 {
		return fmt.Errorf("invalid query_retention: must be non-negative")
	}

	return nil
}

// GetOpenAIClientConfig creates OpenAI client configuration from APIConfig
func (c *AIConfig) GetOpenAIClientConfig() OpenAIConfig {
	return OpenAIConfig{
		APIKey:           c.OpenAI.APIKey,
		BaseURL:          c.OpenAI.BaseURL,
		CompletionModel:  c.OpenAI.CompletionModel,
		EmbeddingModel:   c.OpenAI.EmbeddingModel,
		Timeout:          c.OpenAI.Timeout,
		RateLimit:        c.OpenAI.RateLimit,
		RateLimitWindow:  c.OpenAI.RateLimitWindow,
		Temperature:      c.OpenAI.Temperature,
		MaxTokens:        c.OpenAI.MaxTokens,
		EnableCache:      c.OpenAI.EnableCache,
	}
}

// GetPGVectorConfig creates PGVector client configuration
func (c *AIConfig) GetPGVectorConfig() PGVectorConfig {
	return PGVectorConfig{
		Enabled:       c.PGVector.Enabled,
		DSN:           c.PGVector.DSN,
		CollectionSize: c.PGVector.CollectionSize,
		COSIMDistance: c.PGVector.COSIMDistance,
		TopK:          c.PGVector.TopK,
		Threshold:     c.PGVector.Threshold,
	}
}

// ToEnvMap converts config to environment variable map
func (c *AIConfig) ToEnvMap() map[string]string {
	return map[string]string{
		"AXIOM_AI_OPENAI_ENABLED":         fmt.Sprintf("%v", c.OpenAI.Enabled),
		"AXIOM_AI_OPENAI_BASE_URL":        c.OpenAI.BaseURL,
		"AXIOM_AI_OPENAI_COMPLETION_MODEL": c.OpenAI.CompletionModel,
		"AXIOM_AI_OPENAI_EMBEDDING_MODEL": c.OpenAI.EmbeddingModel,
		"AXIOM_AI_OPENAI_TIMEOUT":         c.OpenAI.Timeout.String(),
		"AXIOM_AI_OPENAI_RATE_LIMIT":      fmt.Sprintf("%d", c.OpenAI.RateLimit),
		"AXIOM_AI_OPENAI_TEMPERATURE":     fmt.Sprintf("%.1f", c.OpenAI.Temperature),
		"AXIOM_AI_OPENAI_MAX_TOKENS":      fmt.Sprintf("%d", c.OpenAI.MaxTokens),
		"AXIOM_AI_PGVECTOR_ENABLED":       fmt.Sprintf("%v", c.PGVector.Enabled),
		"AXIOM_AI_PGVECTOR_COLLECTION_SIZE": fmt.Sprintf("%d", c.PGVector.CollectionSize),
		"AXIOM_AI_ENGINE_DEFAULT_CONTEXT_SIZE": fmt.Sprintf("%d", c.Engine.DefaultContextSize),
		"AXIOM_AI_ENGINE_MAX_CONTEXT_SIZE": fmt.Sprintf("%d", c.Engine.MaxContextSize),
		"AXIOM_AI_ENGINE_RECOMMENDATION_LIMIT": fmt.Sprintf("%d", c.Engine.RecommendationLimit),
		"AXIOM_AI_CONTEXT_QUERY_RETENTION": fmt.Sprintf("%d", c.Context.QueryRetention),
	}
}
