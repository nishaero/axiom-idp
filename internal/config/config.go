package config

import (
	"fmt"
	"os"
	"strconv"
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
	OAuthProvider      string
	OAuthClientID      string
	OAuthClientSecret  string
	OAuthRedirectURL   string

	// MCP
	MCPConfigPath string
	MCPTimeout    time.Duration

	// Session
	SessionSecret string
	SessionMaxAge int

	// CORS
	CORSOrigins []string

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

// NewConfig creates configuration from environment variables
func NewConfig() *Config {
	return &Config{
		Port:              getEnvInt("AXIOM_PORT", 8080),
		Host:              getEnv("AXIOM_HOST", "0.0.0.0"),
		Environment:       getEnv("AXIOM_ENV", "development"),
		LogLevel:          getEnv("AXIOM_LOG_LEVEL", "info"),
		DBDriver:          getEnv("AXIOM_DB_DRIVER", "sqlite3"),
		DBURL:             getEnv("AXIOM_DB_URL", "file:axiom.db"),
		OAuthProvider:     getEnv("AXIOM_OAUTH_PROVIDER", ""),
		OAuthClientID:     getEnv("AXIOM_OAUTH_CLIENT_ID", ""),
		OAuthClientSecret: getEnv("AXIOM_OAUTH_CLIENT_SECRET", ""),
		OAuthRedirectURL:  getEnv("AXIOM_OAUTH_REDIRECT_URL", "http://localhost:8080/auth/callback"),
		MCPConfigPath:     getEnv("AXIOM_MCP_CONFIG", "/etc/axiom/mcp.yaml"),
		MCPTimeout:        getEnvDuration("AXIOM_MCP_TIMEOUT", "30s"),
		SessionSecret:     getEnv("AXIOM_SESSION_SECRET", "dev-secret"),
		SessionMaxAge:     getEnvInt("AXIOM_SESSION_MAX_AGE", 86400),
		CORSOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
		},
		RateLimitRequests: getEnvInt("AXIOM_RATE_LIMIT_REQUESTS", 1000),
		RateLimitWindow:   getEnvDuration("AXIOM_RATE_LIMIT_WINDOW", "1m"),
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

	if c.SessionSecret == "dev-secret" && c.Environment == "production" {
		return fmt.Errorf("session secret must be set in production")
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
