package config

import (
	"testing"
	"time"
)

func TestNewConfigParsesCorsOrigins(t *testing.T) {
	t.Setenv("AXIOM_CORS_ORIGINS", "https://app.example.com, https://admin.example.com")
	t.Setenv("AXIOM_CORS_ALLOW_CREDENTIALS", "true")

	cfg := NewConfig()

	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("Expected 2 origins, got %d", len(cfg.CORSOrigins))
	}

	if !cfg.CORSAllowCredentials {
		t.Fatal("Expected credentials flag to be true")
	}
}

func TestValidateRejectsInvalidLimits(t *testing.T) {
	cfg := &Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 0,
		RateLimitWindow:   0,
		AIBackend:         "local",
		AITimeout:         time.Second,
		AIMaxTokens:       1,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Expected validation error for invalid rate limit settings")
	}
}

func TestValidateAcceptsOllamaConfig(t *testing.T) {
	cfg := &Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		AIBackend:         "ollama",
		AIBaseURL:         "http://127.0.0.1:11434",
		AIModel:           "qwen3.5:9b",
		AITimeout:         30 * time.Second,
		AIMaxTokens:       512,
		JobQueueSize:      32,
		JobWorkerCount:    2,
		JobHistorySize:    50,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Expected Ollama config to validate, got %v", err)
	}
}

func TestValidateAcceptsOpenAICompatibleConfig(t *testing.T) {
	cfg := &Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		AIBackend:         "openai",
		AIBaseURL:         "https://api.openai.com/v1",
		AIAPIKey:          "test-key",
		AIModel:           "gpt-4o-mini",
		AITimeout:         30 * time.Second,
		AIMaxTokens:       512,
		JobQueueSize:      32,
		JobWorkerCount:    2,
		JobHistorySize:    50,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Expected OpenAI-compatible config to validate, got %v", err)
	}
}

func TestValidateRejectsOpenAICompatibleConfigWithoutAPIKey(t *testing.T) {
	cfg := &Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		AIBackend:         "openai",
		AIBaseURL:         "https://api.openai.com/v1",
		AIModel:           "gpt-4o-mini",
		AITimeout:         30 * time.Second,
		AIMaxTokens:       512,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Expected validation error for missing OpenAI-compatible API key")
	}
}

func TestValidateAcceptsLocalConfigWithoutOllamaSettings(t *testing.T) {
	cfg := &Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		AIBackend:         "local",
		AIBaseURL:         "",
		AIModel:           "",
		AITimeout:         30 * time.Second,
		AIMaxTokens:       512,
		JobQueueSize:      32,
		JobWorkerCount:    2,
		JobHistorySize:    50,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Expected local config to validate without Ollama settings, got %v", err)
	}
}

func TestValidateAcceptsPostgresConfig(t *testing.T) {
	cfg := &Config{
		Port:              8080,
		Environment:       "production",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"https://app.example.com"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		DBDriver:          "postgres",
		DBURL:             "postgres://axiom:axiom@localhost:5432/axiom?sslmode=disable",
		AIBackend:         "local",
		AITimeout:         30 * time.Second,
		AIMaxTokens:       512,
		JobQueueSize:      32,
		JobWorkerCount:    2,
		JobHistorySize:    50,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Expected postgres config to validate, got %v", err)
	}
}

func TestNormalizedDBDriver(t *testing.T) {
	cfg := &Config{DBDriver: "sqlite3"}
	if got := cfg.NormalizedDBDriver(); got != "sqlite" {
		t.Fatalf("Expected sqlite3 to normalize to sqlite, got %q", got)
	}

	cfg.DBDriver = "postgresql"
	if got := cfg.NormalizedDBDriver(); got != "postgres" {
		t.Fatalf("Expected postgresql to normalize to postgres, got %q", got)
	}
}
