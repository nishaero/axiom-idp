package server

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/axiom-idp/axiom/internal/auth"
	"github.com/axiom-idp/axiom/internal/config"
	"github.com/sirupsen/logrus"
)

func TestServerHealth(t *testing.T) {
	cfg := &config.Config{
		Port:        8080,
		Environment: "test",
		LogLevel:    "info",
		CORSOrigins: []string{"http://localhost:3000"},
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

func TestServerMiddleware(t *testing.T) {
	cfg := &config.Config{
		Port:        8080,
		Environment: "test",
		LogLevel:    "info",
		CORSOrigins: []string{"http://localhost:3000"},
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test CORS middleware
	req := httptest.NewRequest("OPTIONS", "/api/v1/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	handler := server.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("Expected CORS origin header, got %q", got)
	}
}

func TestAuthMiddleware(t *testing.T) {
	cfg := &config.Config{
		Port:        8080,
		Environment: "test",
		LogLevel:    "info",
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test missing auth header
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", w.Code)
	}

	// Skip auth for login endpoint
	req = httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for login, got %d", w.Code)
	}

	token, err := server.authManager.GenerateTokenWithRoles("user-1", []string{auth.RoleViewer}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate auth token: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with valid token, got %d", w.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("Expected nosniff, got %q", got)
	}
	if got := w.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("Expected HSTS header over TLS")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(1, time.Second)
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/services", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("Expected status 429, got %d", w.Code)
	}
}

func TestAuditMiddleware(t *testing.T) {
	auditor := NewAuditor()
	handler := auditor.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/services", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleViewer}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logs := auditor.GetLogs("user-1", 10)
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}
	if logs[0].Status != "success" {
		t.Fatalf("Expected success audit entry, got %s", logs[0].Status)
	}
}

func TestLoginHandlerIssuesSignedToken(t *testing.T) {
	cfg := &config.Config{
		Port:                 8080,
		Environment:          "test",
		LogLevel:             "info",
		SessionSecret:        "test-secret",
		CORSOrigins:          []string{"http://localhost:3000"},
		CORSAllowCredentials: false,
		CORSMaxAge:           time.Minute,
		RateLimitRequests:    100,
		RateLimitWindow:      time.Minute,
		SessionMaxAge:        86400,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	body := strings.NewReader(`{"user_id":"demo-user","roles":["viewer"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	w := httptest.NewRecorder()
	server.handleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	token, _ := resp["token"].(string)
	if token == "" || !strings.HasPrefix(token, "v1.") {
		t.Fatalf("Expected signed token, got %q", token)
	}
}

func TestHandleAIQueryLocalBackend(t *testing.T) {
	cfg := &config.Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		AIBackend:         "local",
		AITimeout:         5 * time.Second,
		AIMaxTokens:       128,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"release risk for payments-api"}`))
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleViewer}))
	w := httptest.NewRecorder()
	server.handleAIQuery(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["response"] == "" {
		t.Fatal("Expected non-empty AI response")
	}
}

func TestHandleAIQueryOllamaBackend(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			http.NotFound(w, r)
			return
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode ollama request: %v", err)
		}
		if think, ok := payload["think"].(bool); !ok || think {
			t.Fatalf("expected think=false in ollama request, got %v", payload["think"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":"Ollama says this release is ready with caution."}`))
	}))
	defer ollama.Close()

	cfg := &config.Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		AIBackend:         "ollama",
		AIBaseURL:         ollama.URL,
		AIModel:           "qwen3.5:9b",
		AITimeout:         5 * time.Second,
		AIMaxTokens:       128,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"summarize release readiness"}`))
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleEngineer}))
	w := httptest.NewRecorder()
	server.handleAIQuery(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["backend"] != "ollama" {
		t.Fatalf("Expected ollama backend, got %v", resp["backend"])
	}
}

func TestHandleAIQueryFallsBackToLocalWhenOllamaFails(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary failure", http.StatusServiceUnavailable)
	}))
	defer ollama.Close()

	cfg := &config.Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		AIBackend:         "ollama",
		AIBaseURL:         ollama.URL,
		AIModel:           "qwen3.5:9b",
		AITimeout:         5 * time.Second,
		AIMaxTokens:       128,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"summarize release readiness"}`))
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleEngineer}))
	w := httptest.NewRecorder()
	server.handleAIQuery(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["backend"] != "local-fallback" {
		t.Fatalf("Expected local fallback backend, got %v", resp["backend"])
	}
	if resp["response"] == "" {
		t.Fatal("Expected fallback AI response")
	}
}
