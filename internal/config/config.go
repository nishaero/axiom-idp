package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	// Server
	Port        int
	Host        string
	Environment string
	LogLevel    string

	// Database
	DBDriver string
	DBURL    string

	// OAuth2
	OAuthProvider     string
	OAuthClientID     string
	OAuthClientSecret string
	OAuthRedirectURL  string

	// MCP
	MCPConfigPath string
	MCPTimeout    time.Duration

	// Session
	SessionSecret string
	SessionMaxAge int

	// CORS
	CORSOrigins          []string
	CORSAllowCredentials bool
	CORSMaxAge           time.Duration

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   time.Duration

	// AI
	AIBackend   string
	AIBaseURL   string
	AIModel     string
	AITimeout   time.Duration
	AIMaxTokens int
}

// NewConfig creates configuration from environment variables
func NewConfig() *Config {
	return &Config{
		Port:                 getEnvInt("AXIOM_PORT", 8080),
		Host:                 strings.TrimSpace(getEnv("AXIOM_HOST", "0.0.0.0")),
		Environment:          strings.ToLower(strings.TrimSpace(getEnv("AXIOM_ENV", "development"))),
		LogLevel:             strings.ToLower(strings.TrimSpace(getEnv("AXIOM_LOG_LEVEL", "info"))),
		DBDriver:             getEnv("AXIOM_DB_DRIVER", "sqlite3"),
		DBURL:                getEnv("AXIOM_DB_URL", "file:axiom.db"),
		OAuthProvider:        getEnv("AXIOM_OAUTH_PROVIDER", ""),
		OAuthClientID:        getEnv("AXIOM_OAUTH_CLIENT_ID", ""),
		OAuthClientSecret:    getEnv("AXIOM_OAUTH_CLIENT_SECRET", ""),
		OAuthRedirectURL:     getEnv("AXIOM_OAUTH_REDIRECT_URL", "http://localhost:8080/auth/callback"),
		MCPConfigPath:        getEnv("AXIOM_MCP_CONFIG", "/etc/axiom/mcp.yaml"),
		MCPTimeout:           getEnvDuration("AXIOM_MCP_TIMEOUT", "30s"),
		SessionSecret:        getEnv("AXIOM_SESSION_SECRET", "dev-secret"),
		SessionMaxAge:        getEnvInt("AXIOM_SESSION_MAX_AGE", 86400),
		CORSOrigins:          getEnvList("AXIOM_CORS_ORIGINS", []string{"http://localhost:3000", "http://localhost:8080"}),
		CORSAllowCredentials: getEnvBool("AXIOM_CORS_ALLOW_CREDENTIALS", false),
		CORSMaxAge:           getEnvDuration("AXIOM_CORS_MAX_AGE", "10m"),
		RateLimitRequests:    getEnvInt("AXIOM_RATE_LIMIT_REQUESTS", 1000),
		RateLimitWindow:      getEnvDuration("AXIOM_RATE_LIMIT_WINDOW", "1m"),
		AIBackend:            strings.ToLower(strings.TrimSpace(getEnv("AXIOM_AI_BACKEND", "local"))),
		AIBaseURL:            strings.TrimSpace(getEnv("AXIOM_AI_BASE_URL", "http://127.0.0.1:11434")),
		AIModel:              strings.TrimSpace(getEnv("AXIOM_AI_MODEL", "qwen3.5:9b")),
		AITimeout:            getEnvDuration("AXIOM_AI_TIMEOUT", "90s"),
		AIMaxTokens:          getEnvInt("AXIOM_AI_MAX_TOKENS", 768),
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.LogLevel != "debug" && c.LogLevel != "info" && c.LogLevel != "warn" && c.LogLevel != "error" {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	if c.SessionMaxAge <= 0 {
		return fmt.Errorf("session max age must be positive")
	}

	if c.SessionSecret == "dev-secret" && c.Environment == "production" {
		return fmt.Errorf("session secret must be set in production")
	}

	if c.RateLimitRequests <= 0 {
		return fmt.Errorf("rate limit requests must be positive")
	}

	if c.RateLimitWindow <= 0 {
		return fmt.Errorf("rate limit window must be positive")
	}

	if c.AIBackend != "local" && c.AIBackend != "ollama" {
		return fmt.Errorf("ai backend must be one of: local, ollama")
	}

	if c.AITimeout < 1*time.Second {
		return fmt.Errorf("ai timeout must be at least 1 second")
	}

	if c.AIMaxTokens <= 0 {
		return fmt.Errorf("ai max tokens must be positive")
	}

	if c.AIBackend == "ollama" {
		if strings.TrimSpace(c.AIBaseURL) == "" {
			return fmt.Errorf("ai base url must be set when ollama backend is enabled")
		}
		if strings.TrimSpace(c.AIModel) == "" {
			return fmt.Errorf("ai model must be set when ollama backend is enabled")
		}
	}

	if len(c.CORSOrigins) == 0 {
		return fmt.Errorf("at least one CORS origin must be configured")
	}

	for _, origin := range c.CORSOrigins {
		if strings.TrimSpace(origin) == "" {
			return fmt.Errorf("cors origins must not contain empty values")
		}
		if origin == "*" && c.CORSAllowCredentials {
			return fmt.Errorf("wildcard CORS origin cannot be used with credentials")
		}
	}

	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}

func getEnvDuration(key, defaultValue string) time.Duration {
	value := getEnv(key, defaultValue)
	duration, err := time.ParseDuration(value)
	if err != nil {
		durations, _ := time.ParseDuration(defaultValue)
		return durations
	}
	return duration
}

func getEnvBool(key string, defaultValue bool) bool {
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func getEnvList(key string, defaultValue []string) []string {
	value := getEnv(key, "")
	if value == "" {
		out := make([]string, len(defaultValue))
		copy(out, defaultValue)
		return out
	}

	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		origins = append(origins, trimmed)
	}

	if len(origins) == 0 {
		out := make([]string, len(defaultValue))
		copy(out, defaultValue)
		return out
	}

	return origins
}
