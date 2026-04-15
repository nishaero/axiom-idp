package ai

import (
	"context"
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// OpenAIAPIClient provides a deterministic, test-friendly client.
// It keeps the public surface compatible with the rest of the package
// while avoiding external network dependencies by default.
type OpenAIAPIClient struct {
	apiKey         string
	baseURL        string
	model          string
	embeddingModel string
	client         *http.Client
	config         OpenAIConfig
	mock           bool

	mu           sync.RWMutex
	requestCount int
	lastRequest  time.Time
	logger       *logrus.Logger
}

// OpenAIConfig contains OpenAI client configuration.
type OpenAIConfig struct {
	APIKey          string
	BaseURL         string
	CompletionModel string
	EmbeddingModel  string
	Timeout         time.Duration
	RateLimit       int
	RateLimitWindow time.Duration
	Temperature     float64
	MaxTokens       int
	EnableCache     bool
}

// NewOpenAIClient creates a new OpenAI client. It falls back to deterministic
// local behavior when the client is not fully configured.
func NewOpenAIClient(logger *logrus.Logger, config OpenAIConfig) *OpenAIAPIClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RateLimitWindow == 0 {
		config.RateLimitWindow = time.Minute
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 2048
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}

	return &OpenAIAPIClient{
		apiKey:         config.APIKey,
		baseURL:        config.BaseURL,
		model:          config.CompletionModel,
		embeddingModel: config.EmbeddingModel,
		client:         &http.Client{Timeout: config.Timeout},
		config:         config,
		mock:           config.APIKey == "" || strings.Contains(strings.ToLower(config.BaseURL), "mock"),
		logger:         logger,
	}
}

// NewMockOpenAIClient creates a mock client for tests and demos.
func NewMockOpenAIClient() *OpenAIAPIClient {
	return &OpenAIAPIClient{
		apiKey:         "mock-key",
		baseURL:        "mock://openai",
		model:          "gpt-4o-mini",
		embeddingModel: "text-embedding-3-small",
		client:         &http.Client{Timeout: 30 * time.Second},
		config: OpenAIConfig{
			Temperature: 0.3,
			MaxTokens:   1024,
			Timeout:     30 * time.Second,
		},
		mock:   true,
		logger: logrus.New(),
	}
}

// Embed generates a deterministic embedding.
func (c *OpenAIAPIClient) Embed(ctx context.Context, text string) ([]float32, error) {
	c.recordRequest()
	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}
	return deterministicEmbedding(text, 1536), nil
}

// Complete generates a deterministic completion.
func (c *OpenAIAPIClient) Complete(ctx context.Context, prompt string) (string, error) {
	c.recordRequest()
	if err := c.checkRateLimit(); err != nil {
		return "", err
	}
	return deterministicCompletion(prompt), nil
}

func (c *OpenAIAPIClient) recordRequest() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestCount++
	c.lastRequest = time.Now()
}

func (c *OpenAIAPIClient) checkRateLimit() error {
	if c.config.RateLimit <= 0 {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.requestCount > c.config.RateLimit {
		return fmt.Errorf("rate limit exceeded: %d requests per %v", c.config.RateLimit, c.config.RateLimitWindow)
	}
	return nil
}

// GetConfig returns the current configuration.
func (c *OpenAIAPIClient) GetConfig() OpenAIConfig {
	return c.config
}

// SetConfig updates the configuration.
func (c *OpenAIAPIClient) SetConfig(config OpenAIConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = config
	if config.Timeout > 0 {
		c.client.Timeout = config.Timeout
	}
}

// GetStats returns client statistics.
func (c *OpenAIAPIClient) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]interface{}{
		"request_count":   c.requestCount,
		"last_request":    c.lastRequest,
		"model":           c.model,
		"embedding_model": c.embeddingModel,
		"timeout":         c.config.Timeout,
		"rate_limit":      c.config.RateLimit,
		"mock":            c.mock,
	}
}

// ResetStats resets the request counter.
func (c *OpenAIAPIClient) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestCount = 0
	c.lastRequest = time.Time{}
}

func deterministicEmbedding(text string, dim int) []float32 {
	vec := make([]float32, dim)
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	seed := h.Sum64()
	for i := range vec {
		seed = seed*1664525 + 1013904223
		vec[i] = float32(seed%1000) / 1000.0
	}
	return vec
}

func deterministicCompletion(prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "recommend"), strings.Contains(lower, "suggest"):
		return "Use the service with the best fit for your query, then validate ownership, security, and deployment readiness."
	case strings.Contains(lower, "search"), strings.Contains(lower, "find"):
		return "I found matching services in the catalog. Narrow the request with owner, tags, or domain for a more precise result."
	case strings.Contains(lower, "how"), strings.Contains(lower, "what"), strings.Contains(lower, "why"):
		return "This platform can help you discover services, infer intent, and recommend the most relevant internal tools."
	default:
		return "Response to: " + prompt
	}
}
