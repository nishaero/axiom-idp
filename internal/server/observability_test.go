package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/axiom-idp/axiom/internal/config"
	"github.com/sirupsen/logrus"
)

func TestObservabilityEndpointAndMetricsExport(t *testing.T) {
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

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"user_id":"demo-user","roles":["viewer"]}`))
	loginW := httptest.NewRecorder()
	server.router.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusOK {
		t.Fatalf("Expected login to succeed, got %d", loginW.Code)
	}

	var loginResp map[string]interface{}
	if err := json.Unmarshal(loginW.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("Failed to parse login response: %v", err)
	}

	token, _ := loginResp["token"].(string)
	if token == "" {
		t.Fatal("Expected signed token from login flow")
	}

	aiReq := httptest.NewRequest(http.MethodPost, "/api/v1/ai/query", strings.NewReader(`{"query":"List available services"}`))
	aiReq.Header.Set("Authorization", "Bearer "+token)
	aiW := httptest.NewRecorder()
	server.router.ServeHTTP(aiW, aiReq)
	if aiW.Code != http.StatusOK {
		t.Fatalf("Expected AI query to succeed, got %d", aiW.Code)
	}

	observabilityReq := httptest.NewRequest(http.MethodGet, "/api/v1/platform/observability", nil)
	observabilityReq.Header.Set("Authorization", "Bearer "+token)
	observabilityW := httptest.NewRecorder()
	server.router.ServeHTTP(observabilityW, observabilityReq)
	if observabilityW.Code != http.StatusOK {
		t.Fatalf("Expected observability endpoint to succeed, got %d", observabilityW.Code)
	}

	var observabilityResp map[string]interface{}
	if err := json.Unmarshal(observabilityW.Body.Bytes(), &observabilityResp); err != nil {
		t.Fatalf("Failed to parse observability response: %v", err)
	}

	telemetry, ok := observabilityResp["telemetry"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected telemetry snapshot, got %v", observabilityResp)
	}
	if telemetry["http_requests_total"] == nil || telemetry["ai_requests_total"] == nil {
		t.Fatalf("Expected telemetry counters, got %v", telemetry)
	}
	if observabilityResp["metrics_endpoint"] != "/metrics" {
		t.Fatalf("Expected metrics endpoint hint, got %v", observabilityResp["metrics_endpoint"])
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsW := httptest.NewRecorder()
	server.router.ServeHTTP(metricsW, metricsReq)
	if metricsW.Code != http.StatusOK {
		t.Fatalf("Expected metrics endpoint to succeed, got %d", metricsW.Code)
	}

	metricsBody := metricsW.Body.String()
	if !strings.Contains(metricsBody, "axiom_http_requests_total") {
		t.Fatal("Expected Prometheus HTTP request counter in metrics output")
	}
	if !strings.Contains(metricsBody, "axiom_ai_requests_total") {
		t.Fatal("Expected Prometheus AI request counter in metrics output")
	}
	if !strings.Contains(metricsBody, "axiom_audit_events_total") {
		t.Fatal("Expected Prometheus audit counter in metrics output")
	}
}
