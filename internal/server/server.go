package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/axiom-idp/axiom/internal/config"
)

// Server represents the HTTP server
type Server struct {
	config *config.Config
	logger *logrus.Logger
	router *mux.Router
	http   *http.Server
}

// New creates a new server instance
func New(cfg *config.Config, logger *logrus.Logger) (*Server, error) {
	s := &Server{
		config: cfg,
		logger: logger,
		router: mux.NewRouter(),
	}

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	s.http = &http.Server{
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)

	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Catalog routes
	api.HandleFunc("/catalog/services", s.handleListServices).Methods(http.MethodGet)
	api.HandleFunc("/catalog/search", s.handleSearchCatalog).Methods(http.MethodGet)

	// AI routes
	api.HandleFunc("/ai/query", s.handleAIQuery).Methods(http.MethodPost)

	// Auth routes
	api.HandleFunc("/auth/login", s.handleLogin).Methods(http.MethodPost)
	api.HandleFunc("/auth/logout", s.handleLogout).Methods(http.MethodPost)

	// Middleware
	api.Use(s.loggingMiddleware)
	api.Use(s.authMiddleware)
	api.Use(s.corsMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	s.http.Addr = addr
	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// Handler functions
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"services":[]}`))
}

func (s *Server) handleSearchCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"results":[]}`))
}

func (s *Server) handleAIQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"response":"AI feature coming soon"}`))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"token":"","user":{}}`))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"logged out"}`))
}

// Middleware functions
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
		}).Debug("Incoming request")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for certain endpoints
		if r.URL.Path == "/api/v1/auth/login" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth in development mode
		if s.config.Environment == "development" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for auth token
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
