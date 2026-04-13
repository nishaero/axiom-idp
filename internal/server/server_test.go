package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/axiom-idp/axiom/internal/config"
	"github.com/sirupsen/logrus"
)

func TestServerHealth(t *testing.T) {
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
	}

	logger := logrus.New()
	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test CORS middleware
	req := httptest.NewRequest("OPTIONS", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler := server.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
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

	// Skip auth for login endpoint
	req = httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for login, got %d", w.Code)
	}
}
