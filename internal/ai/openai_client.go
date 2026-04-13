package ai

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// OpenAICompletion represents a completion request to OpenAI
type OpenAICompletion struct {
	Model            string  `json:"model"`
	Prompt           string  `json:"prompt"`
	Temperature      float64 `json:"temperature,omitempty"`
	MaxTokens        int     `json:"max_tokens,omitempty"`
	TopP             float64 `json:"top_p,omitempty"`
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64 `json:"presence_penalty,omitempty"`
	Stop             []string `json:"stop,omitempty"`
}

// OpenAICompletionResponse represents a completion response from OpenAI
type OpenAICompletionResponse struct {
	ID      string                  `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created"`
	Model   string                  `json:"model"`
	Choices []CompletionChoice      `json:"choices"`
	Usage   CompletionUsage         `json:"usage"`
	Error   *OpenAIError            `json:"error,omitempty"`
}

// CompletionChoice represents a completion choice
type CompletionChoice struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	Logprobs     string `json:"logprobs"`
	FinishReason string `json:"finish_reason"`
}

// CompletionUsage represents token usage
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIError represents an OpenAI API error
type OpenAIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Param   string `json:"param"`
	Type    string `json:"type"`
}

// EmbeddingRequest represents an embedding request to OpenAI
type EmbeddingRequest struct {
	Model  string   `json:"model"`
	Input  string   `json:"input"`
	User   string   `json:"user,omitempty"`
}

// EmbeddingResponse represents an embedding response from OpenAI
type EmbeddingResponse struct {
	Object string           `json:"object"`
	Data   []EmbeddingData  `json:"data"`
	Model  string           `json:"model"`
	Usage  EmbeddingUsage   `json:"usage"`
	Error  *OpenAIError     `json:"error,omitempty"`
}

// EmbeddingData represents embedding data
type EmbeddingData struct {
	Object   string    `json:"object"`
	Index    int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

// EmbeddingUsage represents embedding token usage
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// OpenAIClient implements OpenAI API client
type OpenAIClient struct {
	apiKey        string
	baseURL       string
	model         string
	embeddingModel string
	client        *http.Client
	config        OpenAIConfig
	mu            sync.RWMutex
	requestCount  int
	lastRequest   time.Time
	logger        *logrus.Logger
}

// OpenAIConfig contains OpenAI client configuration
type OpenAIConfig struct {
	APIKey           string
	BaseURL          string
	CompletionModel  string
	EmbeddingModel   string
	Timeout          time.Duration
	RateLimit        int
	RateLimitWindow  time.Duration
	Temperature      float64
	MaxTokens        int
	EnableCache      bool
}

// NewOpenAIClient creates a new OpenAI API client
func NewOpenAIClient(logger *logrus.Logger, config OpenAIConfig) *OpenAIClient {
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

	return &OpenAIClient{
		apiKey:         config.APIKey,
		baseURL:        config.BaseURL,
		model:          config.CompletionModel,
		embeddingModel: config.EmbeddingModel,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		config:         config,
		logger:         logger.WithField("component", "openai_client"),
		requestCount:   0,
		lastRequest:    time.Time{},
	}
}

// Embed generates an embedding vector for the given text
func (c *OpenAIClient) Embed(ctx context.Context, text string) ([]float32, error) {
	c.mu.Lock()
	c.requestCount++
	c.lastRequest = time.Now()
	c.mu.Unlock()

	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/embeddings", c.baseURL)

	request := EmbeddingRequest{
		Model: c.embeddingModel,
		Input: text,
	}

	if c.config.EnableCache {
		// Check cache first (implementation would go here)
		// For now, skip caching
		_ = request
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	response, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("embedding API error (status %d): %s", response.StatusCode, string(body))
	}

	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(response.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if embeddingResp.Error != nil {
		return nil, fmt.Errorf("embedding error: %s", embeddingResp.Error.Message)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, errors.New("no embedding data returned")
	}

	return embeddingResp.Data[0].Embedding, nil
}

// Complete sends a completion request to OpenAI
func (c *OpenAIClient) Complete(ctx context.Context, prompt string) (string, error) {
	c.mu.Lock()
	c.requestCount++
	c.lastRequest = time.Now()
	c.mu.Unlock()

	if err := c.checkRateLimit(); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/completions", c.baseURL)

	request := OpenAICompletion{
		Model:         c.model,
		Prompt:        prompt,
		Temperature:   c.config.Temperature,
		MaxTokens:     c.config.MaxTokens,
		FrequencyPenalty: 0.0,
		PresencePenalty: 0.0,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal completion request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create completion request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	response, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("completion request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return "", fmt.Errorf("completion API error (status %d): %s", response.StatusCode, string(body))
	}

	var completionResp OpenAICompletionResponse
	if err := json.NewDecoder(response.Body).Decode(&completionResp); err != nil {
		return "", fmt.Errorf("failed to decode completion response: %w", err)
	}

	if completionResp.Error != nil {
		return "", fmt.Errorf("completion error: %s", completionResp.Error.Message)
	}

	if len(completionResp.Choices) == 0 {
		return "", errors.New("no completion choices returned")
	}

	return completionResp.Choices[0].Text, nil
}

// checkRateLimit checks if we've exceeded the rate limit
func (c *OpenAIClient) checkRateLimit() error {
	if c.config.RateLimit == 0 {
		return nil
	}

	c.mu.RLock()
	count := c.requestCount
	c.mu.RUnlock()

	if count >= c.config.RateLimit {
		return fmt.Errorf("rate limit exceeded: %d requests per %v", c.config.RateLimit, c.config.RateLimitWindow)
	}

	return nil
}

// GetConfig returns the current configuration
func (c *OpenAIClient) GetConfig() OpenAIConfig {
	return c.config
}

// SetConfig updates the configuration
func (c *OpenAIClient) SetConfig(config OpenAIConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = config
	if config.Timeout > 0 {
		c.client.Timeout = config.Timeout
	}
}

// GetStats returns client statistics
func (c *OpenAIClient) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"request_count": c.requestCount,
		"last_request":  c.lastRequest,
		"model":         c.model,
		"embedding_model": c.embeddingModel,
		"timeout":       c.config.Timeout,
		"rate_limit":    c.config.RateLimit,
	}
}

// ResetStats resets the request counter
func (c *OpenAIClient) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestCount = 0
	c.lastRequest = time.Time{}
}

// NewMockOpenAIClient creates a mock OpenAI client for testing
func NewMockOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		apiKey:         "mock-key",
		baseURL:        "https://api.openai.com/v1",
		model:          "gpt-3.5-turbo",
		embeddingModel: "text-embedding-ada-002",
		client:         &http.Client{Timeout: 30 * time.Second},
		config: OpenAIConfig{
			Temperature:   0.7,
			MaxTokens:     2048,
			Timeout:       30 * time.Second,
		},
		logger: logrus.New(),
	}
}

// Embed returns mock embeddings (fixed vector for consistency)
func (c *OpenAIClient) Embed(ctx context.Context, text string) ([]float32, error) {
	// Generate deterministic vector based on text hash
	vector := make([]float32, 1536)
	hash := hashString(text)

	for i := range vector {
		vector[i] = float32((hash+i*7)%1000) / 1000.0
	}

	return vector, nil
}

// Complete returns mock completions based on the prompt
func (c *OpenAIClient) Complete(ctx context.Context, prompt string) (string, error) {
	// Simple mock completion logic
	lowerPrompt := fmt.Sprintf(" %s ", prompt)

	if containsSubstring(lowerPrompt, "recommend") || containsSubstring(lowerPrompt, "suggest") {
		return "Based on the criteria, I recommend using a microservices architecture with containerized deployments. This approach provides scalability, isolation, and easier maintenance.", nil
	}

	if containsSubstring(lowerPrompt, "search") || containsSubstring(lowerPrompt, "find") {
		return "The service catalog contains multiple matching services. Please specify additional filters to narrow down the results.", nil
	}

	if containsSubstring(lowerPrompt, "how") || containsSubstring(lowerPrompt, "what") || containsSubstring(lowerPrompt, "why") {
		return "This is a helpful response to your question about the Axiom IDP platform.", nil
	}

	return fmt.Sprintf("Response to: %s", prompt), nil
}

// hashString generates a simple hash of a string
func hashString(s string) uint32 {
	var hash uint32 = 5381
	for _, c := range s {
		hash = (hash * 31) ^ uint32(c)
	}
	return hash
}

// containsSubstring checks if the long string contains the short string
func containsSubstring(long, short string) bool {
	for i := 0; i <= len(long)-len(short); i++ {
		if long[i:i+len(short)] == short {
			return true
		}
	}
	return false
}
