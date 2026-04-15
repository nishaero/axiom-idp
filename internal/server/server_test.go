package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/axiom-idp/axiom/internal/auth"
	"github.com/axiom-idp/axiom/internal/config"
	"github.com/gorilla/mux"
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

func TestServerReady(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected ready endpoint to return 200, got %d", w.Code)
	}
}

func TestProductionSQLiteReadinessIsDegraded(t *testing.T) {
	cfg := &config.Config{
		Port:              8080,
		Environment:       "production",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		DBDriver:          "sqlite3",
		DBURL:             "file:axiom.db",
		AIBackend:         "local",
		AITimeout:         5 * time.Second,
		AIMaxTokens:       128,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected degraded readiness to stay HTTP 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse ready response: %v", err)
	}
	if resp["status"] != statusDegraded {
		t.Fatalf("Expected degraded readiness status, got %v", resp["status"])
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

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected /metrics to bypass rate limiting, got %d", w.Code)
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
	if resp["intent"] != "release_readiness" {
		t.Fatalf("Expected release_readiness intent, got %v", resp["intent"])
	}
	if resp["focus_service"] == nil {
		t.Fatal("Expected focus service analysis in structured response")
	}
	actions, ok := resp["next_steps"].([]interface{})
	if !ok || len(actions) == 0 {
		t.Fatal("Expected structured AI actions")
	}
}

func TestHandleAIQueryReleaseBrief(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"generate a release brief for payments-api"}`))
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

	if resp["intent"] != "release_brief" {
		t.Fatalf("Expected release_brief intent, got %v", resp["intent"])
	}
	if resp["response"] == "" {
		t.Fatal("Expected non-empty release brief response")
	}
}

func TestHandleAIQueryOllamaBackend(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode ollama request: %v", err)
		}
		if model := payload["model"]; model != "qwen3.5:9b" {
			t.Fatalf("expected ollama model qwen3.5:9b, got %v", model)
		}
		messages, ok := payload["messages"].([]interface{})
		if !ok || len(messages) < 2 {
			t.Fatalf("expected chat messages in ollama-compatible request, got %T", payload["messages"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Ollama says this release is ready with caution."}}]}`))
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
	if resp["generated_text"] == "" {
		t.Fatal("Expected generated_text in response")
	}
	if resp["portfolio"] == nil {
		t.Fatal("Expected structured portfolio intelligence")
	}
}

func TestHandleAIQueryOpenAICompatibleBackend(t *testing.T) {
	openai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if authz := r.Header.Get("Authorization"); authz != "Bearer test-openai-key" {
			t.Fatalf("expected bearer token auth header, got %q", authz)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode openai-compatible request: %v", err)
		}
		if model := payload["model"]; model != "gpt-4o-mini" {
			t.Fatalf("expected model gpt-4o-mini, got %v", model)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Cloud model says the release should remain on watch."}}]}`))
	}))
	defer openai.Close()

	cfg := &config.Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		AIBackend:         "openai",
		AIBaseURL:         openai.URL,
		AIAPIKey:          "test-openai-key",
		AIModel:           "gpt-4o-mini",
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

	if resp["backend"] != "openai" {
		t.Fatalf("Expected openai backend, got %v", resp["backend"])
	}
	if resp["generated_text"] == "" {
		t.Fatal("Expected generated_text in response")
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

func TestCatalogOverviewEndpoint(t *testing.T) {
	cfg := &config.Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/overview", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleViewer}))
	w := httptest.NewRecorder()
	server.handleCatalogOverview(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	overview, ok := resp["overview"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected nested overview payload, got %v", resp)
	}
	portfolio, ok := resp["portfolio"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected nested portfolio payload, got %v", resp)
	}
	if overview["total_services"] == nil || overview["release_readiness"] == nil {
		t.Fatalf("Expected overview metrics, got %v", resp)
	}
	if portfolio["total_services"] == nil || portfolio["risk_level"] == nil {
		t.Fatalf("Expected portfolio metrics, got %v", resp)
	}
}

func TestCatalogServicesIncludeStructuredIntelligence(t *testing.T) {
	cfg := &config.Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/services", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleViewer}))
	w := httptest.NewRecorder()
	server.handleListServices(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	services, ok := resp["services"].([]interface{})
	if !ok || len(services) == 0 {
		t.Fatalf("Expected services collection, got %v", resp["services"])
	}

	first, ok := services[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected service object, got %T", services[0])
	}
	if first["intelligence"] == nil {
		t.Fatal("Expected structured intelligence on catalog service")
	}
}

func TestServiceInsightEndpoint(t *testing.T) {
	cfg := &config.Config{
		Port:              8080,
		Environment:       "test",
		LogLevel:          "info",
		SessionSecret:     "test-secret",
		SessionMaxAge:     86400,
		CORSOrigins:       []string{"http://localhost:3000"},
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/services/svc-data", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "svc-data"})
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleViewer}))
	w := httptest.NewRecorder()
	server.handleServiceInsight(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	insight, ok := resp["insight"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected insight object, got %v", resp["insight"])
	}
	readiness, ok := insight["release_readiness"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected nested release_readiness insight, got %v", insight["release_readiness"])
	}
	if readiness["state"] != "blocked" {
		t.Fatalf("Expected blocked insight, got %v", readiness["state"])
	}

	brief, ok := resp["brief"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected release brief, got %v", resp["brief"])
	}
	if brief["next_best_action"] == "" {
		t.Fatal("Expected next best action in release brief")
	}
	if brief["portfolio_context"] == "" {
		t.Fatal("Expected portfolio context in release brief")
	}
}

type fakeDeploymentManager struct {
	applyRecord  *deploymentRecord
	statusRecord *deploymentRecord
	applyErr     error
	statusErr    error
}

func (f *fakeDeploymentManager) Apply(_ context.Context, _ deploymentApplyRequest) (*deploymentRecord, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return f.applyRecord, nil
}

func (f *fakeDeploymentManager) Status(_ context.Context, _, _ string) (*deploymentRecord, error) {
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	return f.statusRecord, nil
}

type fakeGitOpsManager struct {
	deployRecord *deploymentRecord
	statusRecord *deploymentRecord
	infraRecord  *infrastructureRecord
	deployErr    error
	statusErr    error
	infraErr     error
}

func (f *fakeGitOpsManager) ApplyArgoCDDeployment(_ context.Context, _ deploymentApplyRequest) (*deploymentRecord, error) {
	if f.deployErr != nil {
		return nil, f.deployErr
	}
	return f.deployRecord, nil
}

func (f *fakeGitOpsManager) ArgoCDDeploymentStatus(_ context.Context, _, _ string) (*deploymentRecord, error) {
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	return f.statusRecord, nil
}

func (f *fakeGitOpsManager) ApplyInfrastructure(_ context.Context, _ infrastructureApplyRequest) (*infrastructureRecord, error) {
	if f.infraErr != nil {
		return nil, f.infraErr
	}
	return f.infraRecord, nil
}

func (f *fakeGitOpsManager) TerraformInfrastructureStatus(_ context.Context, _ string) (*infrastructureRecord, error) {
	if f.infraErr != nil {
		return nil, f.infraErr
	}
	return f.infraRecord, nil
}

func waitForAsyncJobStatus(t *testing.T, server *Server, jobID string, wantStatus string) *asyncJob {
	t.Helper()
	if server == nil || server.jobs == nil {
		t.Fatal("Expected server job manager")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		job, ok := server.jobs.Get(jobID)
		if ok && job != nil && job.Status == wantStatus {
			return job
		}
		if time.Now().After(deadline) {
			if job, ok := server.jobs.Get(jobID); ok && job != nil {
				t.Fatalf("Timed out waiting for job %s to reach %s, last status %s", jobID, wantStatus, job.Status)
			}
			t.Fatalf("Timed out waiting for job %s to reach %s", jobID, wantStatus)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestHandleApplyDeploymentEndpoint(t *testing.T) {
	cfg := &config.Config{
		Port:                     8080,
		Environment:              "test",
		LogLevel:                 "info",
		SessionSecret:            "test-secret",
		SessionMaxAge:            86400,
		CORSOrigins:              []string{"http://localhost:3000"},
		RateLimitRequests:        100,
		RateLimitWindow:          time.Minute,
		KubectlPath:              "kubectl",
		KubernetesNamespace:      "axiom-apps",
		KubernetesApplyTimeout:   5 * time.Second,
		KubernetesRolloutTimeout: 5 * time.Second,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.deployer = &fakeDeploymentManager{
		applyRecord: &deploymentRecord{
			Name:              "demo-web",
			Namespace:         "axiom-apps",
			Image:             "nginx:1.27-alpine",
			Replicas:          1,
			ReadyReplicas:     1,
			AvailableReplicas: 1,
			Phase:             "ready",
			ServiceType:       "ClusterIP",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/applications", strings.NewReader(`{"name":"demo-web","image":"nginx:1.27-alpine"}`))
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleEngineer}))
	w := httptest.NewRecorder()
	server.handleApplyDeployment(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected 202, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp["deployment"] == nil {
		t.Fatal("Expected queued deployment payload")
	}
	if resp["job"] == nil {
		t.Fatal("Expected queued job payload")
	}

	job := resp["job"].(map[string]interface{})
	if job["status"] != "queued" {
		t.Fatalf("Expected queued job status, got %v", job["status"])
	}
	jobID, _ := job["id"].(string)
	finalJob := waitForAsyncJobStatus(t, server, jobID, "succeeded")
	if finalJob.Deployment == nil || finalJob.Deployment.Phase != "ready" {
		t.Fatalf("Expected completed deployment result, got %+v", finalJob.Deployment)
	}
}

func TestHandleDeploymentStatusEndpoint(t *testing.T) {
	cfg := &config.Config{
		Port:                     8080,
		Environment:              "test",
		LogLevel:                 "info",
		SessionSecret:            "test-secret",
		SessionMaxAge:            86400,
		CORSOrigins:              []string{"http://localhost:3000"},
		RateLimitRequests:        100,
		RateLimitWindow:          time.Minute,
		KubectlPath:              "kubectl",
		KubernetesNamespace:      "axiom-apps",
		KubernetesApplyTimeout:   5 * time.Second,
		KubernetesRolloutTimeout: 5 * time.Second,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.deployer = &fakeDeploymentManager{
		statusRecord: &deploymentRecord{
			Name:              "demo-web",
			Namespace:         "axiom-apps",
			Image:             "nginx:1.27-alpine",
			Replicas:          1,
			ReadyReplicas:     1,
			AvailableReplicas: 1,
			Phase:             "ready",
			ServiceType:       "ClusterIP",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/applications/axiom-apps/demo-web", nil)
	req = mux.SetURLVars(req, map[string]string{"namespace": "axiom-apps", "name": "demo-web"})
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleViewer}))
	w := httptest.NewRecorder()
	server.handleDeploymentStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestHandleAIQueryDeploymentApply(t *testing.T) {
	cfg := &config.Config{
		Port:                     8080,
		Environment:              "test",
		LogLevel:                 "info",
		SessionSecret:            "test-secret",
		SessionMaxAge:            86400,
		CORSOrigins:              []string{"http://localhost:3000"},
		RateLimitRequests:        100,
		RateLimitWindow:          time.Minute,
		AIBackend:                "local",
		AITimeout:                5 * time.Second,
		AIMaxTokens:              128,
		KubectlPath:              "kubectl",
		KubernetesNamespace:      "axiom-apps",
		KubernetesApplyTimeout:   5 * time.Second,
		KubernetesRolloutTimeout: 5 * time.Second,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.deployer = &fakeDeploymentManager{
		applyRecord: &deploymentRecord{
			Name:              "demo-web",
			Namespace:         "axiom-apps",
			Image:             "nginx:1.27-alpine",
			Replicas:          1,
			ReadyReplicas:     1,
			AvailableReplicas: 1,
			Phase:             "ready",
			ServiceType:       "ClusterIP",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"deploy app demo-web using nginx:1.27-alpine in namespace axiom-apps"}`))
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleEngineer}))
	w := httptest.NewRecorder()
	server.handleAIQuery(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected 202, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp["intent"] != "deployment_apply" {
		t.Fatalf("Expected deployment_apply intent, got %v", resp["intent"])
	}
	if resp["deployment"] == nil {
		t.Fatal("Expected queued deployment payload")
	}
	if resp["job"] == nil {
		t.Fatal("Expected queued job payload")
	}
	actionPlan, ok := resp["action_plan"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected action plan payload, got %T", resp["action_plan"])
	}
	if actionPlan["mode"] != "delivery" {
		t.Fatalf("Expected delivery action plan mode, got %v", actionPlan["mode"])
	}

	job := resp["job"].(map[string]interface{})
	if job["status"] != "queued" {
		t.Fatalf("Expected queued job status, got %v", job["status"])
	}
	jobID, _ := job["id"].(string)
	finalJob := waitForAsyncJobStatus(t, server, jobID, "succeeded")
	if finalJob.Deployment == nil || finalJob.Deployment.Name != "demo-web" {
		t.Fatalf("Expected completed deployment result, got %+v", finalJob.Deployment)
	}
}

func TestHandleAIQueryDeploymentApplyForbidden(t *testing.T) {
	cfg := &config.Config{
		Port:                     8080,
		Environment:              "test",
		LogLevel:                 "info",
		SessionSecret:            "test-secret",
		SessionMaxAge:            86400,
		CORSOrigins:              []string{"http://localhost:3000"},
		RateLimitRequests:        100,
		RateLimitWindow:          time.Minute,
		AIBackend:                "local",
		AITimeout:                5 * time.Second,
		AIMaxTokens:              128,
		KubectlPath:              "kubectl",
		KubernetesNamespace:      "axiom-apps",
		KubernetesApplyTimeout:   5 * time.Second,
		KubernetesRolloutTimeout: 5 * time.Second,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.deployer = &fakeDeploymentManager{
		applyErr: errors.New("should not be called"),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"deploy app demo-web using nginx:1.27-alpine"}`))
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleViewer}))
	w := httptest.NewRecorder()
	server.handleAIQuery(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected 403, got %d", w.Code)
	}
}

func TestHandleAIQueryDeploymentStatus(t *testing.T) {
	cfg := &config.Config{
		Port:                     8080,
		Environment:              "test",
		LogLevel:                 "info",
		SessionSecret:            "test-secret",
		SessionMaxAge:            86400,
		CORSOrigins:              []string{"http://localhost:3000"},
		RateLimitRequests:        100,
		RateLimitWindow:          time.Minute,
		AIBackend:                "local",
		AITimeout:                5 * time.Second,
		AIMaxTokens:              128,
		KubectlPath:              "kubectl",
		KubernetesNamespace:      "axiom-apps",
		KubernetesApplyTimeout:   5 * time.Second,
		KubernetesRolloutTimeout: 5 * time.Second,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.deployer = &fakeDeploymentManager{
		statusRecord: &deploymentRecord{
			Name:              "demo-web",
			Namespace:         "axiom-apps",
			Image:             "nginx:1.27-alpine",
			Replicas:          1,
			ReadyReplicas:     1,
			AvailableReplicas: 1,
			Phase:             "ready",
			Message:           "demo-web has 1/1 ready replicas",
			ServiceType:       "ClusterIP",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"deployment status demo-web in namespace axiom-apps"}`))
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
	if resp["intent"] != "deployment_status" {
		t.Fatalf("Expected deployment_status intent, got %v", resp["intent"])
	}
}

func TestHandleAIQueryArgoCDDeploymentApply(t *testing.T) {
	cfg := &config.Config{
		Port:                     8080,
		Environment:              "test",
		LogLevel:                 "info",
		SessionSecret:            "test-secret",
		SessionMaxAge:            86400,
		CORSOrigins:              []string{"http://localhost:3000"},
		RateLimitRequests:        100,
		RateLimitWindow:          time.Minute,
		AIBackend:                "local",
		AITimeout:                5 * time.Second,
		AIMaxTokens:              128,
		KubectlPath:              "kubectl",
		KubernetesNamespace:      "axiom-apps",
		KubernetesApplyTimeout:   5 * time.Second,
		KubernetesRolloutTimeout: 5 * time.Second,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.gitops = &fakeGitOpsManager{
		deployRecord: &deploymentRecord{
			Name:            "demo-web",
			Namespace:       "axiom-apps",
			Image:           "nginx:1.27-alpine",
			Replicas:        1,
			ReadyReplicas:   1,
			Phase:           "ready",
			Delivery:        "github-argocd",
			Revision:        "axiom-ai/deploy-demo-web",
			ApplicationName: "demo-web",
			SyncStatus:      "Synced",
			HealthStatus:    "Healthy",
			ExecutionState:  "ready",
			ExecutionPlan: &executionPlan{
				Intent:    "deployment_apply_argocd",
				Provider:  "argocd",
				Route:     "github-argocd-kubernetes",
				Mode:      "controller-backed",
				Supported: true,
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"deploy app demo-web using nginx:1.27-alpine via argocd from github in namespace axiom-apps"}`))
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleEngineer}))
	w := httptest.NewRecorder()
	server.handleAIQuery(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected 202, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp["intent"] != "deployment_apply_argocd" {
		t.Fatalf("Expected deployment_apply_argocd intent, got %v", resp["intent"])
	}
	if resp["job"] == nil {
		t.Fatal("Expected queued job payload")
	}
	executionPlan, ok := resp["execution_plan"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected execution plan payload, got %T", resp["execution_plan"])
	}
	if executionPlan["route"] != "github-argocd-kubernetes" {
		t.Fatalf("Expected Argo CD route, got %v", executionPlan["route"])
	}
	actionPlan, ok := resp["action_plan"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected action plan payload, got %T", resp["action_plan"])
	}
	if actionPlan["title"] != "AI-guided GitOps rollout" {
		t.Fatalf("Expected GitOps action plan title, got %v", actionPlan["title"])
	}

	job := resp["job"].(map[string]interface{})
	if job["status"] != "queued" {
		t.Fatalf("Expected queued job status, got %v", job["status"])
	}
	jobID, _ := job["id"].(string)
	finalJob := waitForAsyncJobStatus(t, server, jobID, "succeeded")
	if finalJob.Deployment == nil || finalJob.Deployment.Delivery != "github-argocd" {
		t.Fatalf("Expected completed Argo CD deployment result, got %+v", finalJob.Deployment)
	}
}

func TestHandleAIQueryArgoCDDeploymentStatus(t *testing.T) {
	cfg := &config.Config{
		Port:                     8080,
		Environment:              "test",
		LogLevel:                 "info",
		SessionSecret:            "test-secret",
		SessionMaxAge:            86400,
		CORSOrigins:              []string{"http://localhost:3000"},
		RateLimitRequests:        100,
		RateLimitWindow:          time.Minute,
		AIBackend:                "local",
		AITimeout:                5 * time.Second,
		AIMaxTokens:              128,
		KubectlPath:              "kubectl",
		KubernetesNamespace:      "axiom-apps",
		KubernetesApplyTimeout:   5 * time.Second,
		KubernetesRolloutTimeout: 5 * time.Second,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.gitops = &fakeGitOpsManager{
		statusRecord: &deploymentRecord{
			Name:            "demo-web",
			Namespace:       "axiom-apps",
			Phase:           "ready",
			SyncStatus:      "Synced",
			HealthStatus:    "Healthy",
			ApplicationName: "demo-web",
			Message:         "Deployment synced and healthy",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"argocd deployment status demo-web in namespace axiom-apps from github"}`))
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
	if resp["intent"] != "deployment_status_argocd" {
		t.Fatalf("Expected deployment_status_argocd intent, got %v", resp["intent"])
	}
}

func TestHandleAIQueryInfrastructureTerraform(t *testing.T) {
	cfg := &config.Config{
		Port:                     8080,
		Environment:              "test",
		LogLevel:                 "info",
		SessionSecret:            "test-secret",
		SessionMaxAge:            86400,
		CORSOrigins:              []string{"http://localhost:3000"},
		RateLimitRequests:        100,
		RateLimitWindow:          time.Minute,
		AIBackend:                "local",
		AITimeout:                5 * time.Second,
		AIMaxTokens:              128,
		KubectlPath:              "kubectl",
		KubernetesNamespace:      "axiom-apps",
		KubernetesApplyTimeout:   5 * time.Second,
		KubernetesRolloutTimeout: 5 * time.Second,
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.gitops = &fakeGitOpsManager{
		infraRecord: &infrastructureRecord{
			Name:            "platform-lab",
			Provider:        "terraform",
			TargetNamespace: "platform-lab",
			Phase:           "ready",
			Message:         "Terraform infrastructure request for platform-lab executed through Argo CD.",
			Executed:        true,
			ApplicationName: "infra-platform-lab",
			SyncStatus:      "Synced",
			HealthStatus:    "Healthy",
			ExecutionState:  "ready",
			ExecutionPlan: &executionPlan{
				Intent:    "infrastructure_apply_terraform",
				Provider:  "terraform",
				Route:     "github-argocd-terraform-job",
				Mode:      "controller-backed",
				Supported: true,
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"provision infrastructure namespace platform-lab using terraform"}`))
	req = req.WithContext(auth.ContextWithUser(req.Context(), "user-1", []string{auth.RoleEngineer}))
	w := httptest.NewRecorder()
	server.handleAIQuery(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected 202, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp["intent"] != "infrastructure_apply_terraform" {
		t.Fatalf("Expected infrastructure_apply_terraform intent, got %v", resp["intent"])
	}
	if resp["job"] == nil {
		t.Fatal("Expected queued job payload")
	}
	executionPlan, ok := resp["execution_plan"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected execution plan payload, got %T", resp["execution_plan"])
	}
	if executionPlan["route"] != "github-argocd-terraform-job" {
		t.Fatalf("Expected Terraform route, got %v", executionPlan["route"])
	}
	actionPlan, ok := resp["action_plan"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected action plan payload, got %T", resp["action_plan"])
	}
	if actionPlan["mode"] != "infrastructure" {
		t.Fatalf("Expected infrastructure action plan mode, got %v", actionPlan["mode"])
	}

	job := resp["job"].(map[string]interface{})
	if job["status"] != "queued" {
		t.Fatalf("Expected queued job status, got %v", job["status"])
	}
	jobID, _ := job["id"].(string)
	finalJob := waitForAsyncJobStatus(t, server, jobID, "succeeded")
	if finalJob.Infrastructure == nil || finalJob.Infrastructure.Provider != "terraform" {
		t.Fatalf("Expected completed terraform infrastructure result, got %+v", finalJob.Infrastructure)
	}
}

func TestDeploymentNormalizeAndPlan(t *testing.T) {
	req, err := normalizeDeploymentRequest(deploymentApplyRequest{
		Name:      "Demo Web",
		Namespace: "Apps",
		Image:     "nginx:1.27-alpine",
		Delivery:  "GitHub-Argocd",
	}, "axiom-default")
	if err != nil {
		t.Fatalf("Expected normalized deployment request, got error: %v", err)
	}

	if req.Delivery != "argocd" {
		t.Fatalf("Expected argocd delivery, got %q", req.Delivery)
	}
	plan := newDeploymentExecutionPlan("deployment_apply_argocd", req)
	if plan.Route != "github-argocd-kubernetes" {
		t.Fatalf("Expected Argo CD route, got %q", plan.Route)
	}
	if plan.Mode != "controller-backed" {
		t.Fatalf("Expected controller-backed mode, got %q", plan.Mode)
	}
}

func TestInfrastructureCrossplanePlanAndBundle(t *testing.T) {
	req, err := normalizeInfrastructureRequest(infrastructureApplyRequest{
		Name:            "platform-lab",
		Provider:        "Crossplane",
		TargetNamespace: "platform-lab",
		Inputs: map[string]string{
			"region": "eu-central-1",
			"size":   "small",
		},
	})
	if err != nil {
		t.Fatalf("Expected normalized infrastructure request, got error: %v", err)
	}

	plan := newInfrastructureExecutionPlan("infrastructure_apply_crossplane", req)
	if plan.Route != "github-argocd-crossplane-controller" {
		t.Fatalf("Expected Crossplane route, got %q", plan.Route)
	}
	if plan.Mode != "staged" {
		t.Fatalf("Expected staged mode, got %q", plan.Mode)
	}

	files := renderCrossplaneInfraFiles(req)
	if got := files["README.md"]; !strings.Contains(got, "Inputs:") {
		t.Fatalf("Expected Crossplane README to include inputs, got %q", got)
	}
	if got := files["claim.yaml"]; !strings.Contains(got, "region: \"eu-central-1\"") {
		t.Fatalf("Expected Crossplane claim to include inputs, got %q", got)
	}
}
