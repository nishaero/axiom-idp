package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/axiom-idp/axiom/internal/auth"
	"github.com/axiom-idp/axiom/internal/config"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type demoService struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

var demoCatalog = []demoService{
	{ID: "svc-payments", Name: "payments-api", Owner: "platform", Status: "healthy", Description: "Internal payment orchestration service"},
	{ID: "svc-auth", Name: "identity-gateway", Owner: "security", Status: "healthy", Description: "Authentication and authorization gateway"},
	{ID: "svc-orders", Name: "orders-worker", Owner: "commerce", Status: "degraded", Description: "Event-driven order processing worker"},
}

// Server represents the HTTP server.
type Server struct {
	config      *config.Config
	logger      *logrus.Logger
	router      *mux.Router
	http        *http.Server
	authManager *auth.Manager
	rbac        *auth.RBAC
	auditor     *Auditor
	rateLimiter *RateLimiter
	aiBackend   aiBackend
}

// New creates a new server instance.
func New(cfg *config.Config, logger *logrus.Logger) (*Server, error) {
	s := &Server{
		config:      cfg,
		logger:      logger,
		router:      mux.NewRouter(),
		authManager: auth.NewManager(cfg.SessionSecret),
		rbac:        auth.NewRBAC(),
		auditor:     NewAuditor(),
		rateLimiter: NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow),
		aiBackend:   newAIBackend(cfg, logger),
	}

	s.setupRoutes()

	writeTimeout := 15 * time.Second
	if s.config != nil && s.config.AITimeout > 0 {
		candidate := s.config.AITimeout + (15 * time.Second)
		if candidate > writeTimeout {
			writeTimeout = candidate
		}
	}

	s.http = &http.Server{
		Handler:           s.router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	return s, nil
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	s.router.Use(SecurityHeaders)
	s.router.Use(s.corsMiddleware)
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.rateLimiter.Middleware)

	s.router.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)

	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.authMiddleware)
	api.Use(s.auditor.Middleware)

	api.Handle("/catalog/services", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleListServices))).Methods(http.MethodGet)
	api.Handle("/catalog/search", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleSearchCatalog))).Methods(http.MethodGet)

	api.HandleFunc("/ai/query", s.handleAIQuery).Methods(http.MethodPost)

	api.HandleFunc("/auth/login", s.handleLogin).Methods(http.MethodPost)
	api.HandleFunc("/auth/logout", s.handleLogout).Methods(http.MethodPost)
}

// Start starts the HTTP server.
func (s *Server) Start(addr string) error {
	s.http.Addr = addr
	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"services": demoCatalog})
}

func (s *Server) handleSearchCatalog(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("query"))
	}

	results := make([]demoService, 0, len(demoCatalog))
	for _, service := range demoCatalog {
		if query == "" || strings.Contains(strings.ToLower(service.Name), strings.ToLower(query)) || strings.Contains(strings.ToLower(service.Description), strings.ToLower(query)) {
			results = append(results, service)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"query":   query,
		"results": results,
	})
}

func (s *Server) handleAIQuery(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Query string `json:"query"`
		Text  string `json:"text"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	if strings.TrimSpace(payload.Query) == "" {
		payload.Query = strings.TrimSpace(payload.Text)
	}
	if strings.TrimSpace(payload.Query) == "" {
		payload.Query = strings.TrimSpace(r.URL.Query().Get("q"))
	}
	if strings.TrimSpace(payload.Query) == "" {
		s.writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	roles := auth.RolesFromContext(r.Context())
	ctx, cancel := newAIRequestContext(r.Context(), s.config.AITimeout)
	defer cancel()
	answer, source := queryAI(ctx, s.aiBackend, payload.Query, demoCatalog, userID, roles, s.logger)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"response": answer,
		"backend":  source,
		"query":    payload.Query,
		"user_id":  userID,
		"roles":    roles,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string   `json:"user_id"`
		Name      string   `json:"name"`
		Email     string   `json:"email"`
		Roles     []string `json:"roles"`
		ExpiresIn string   `json:"expires_in"`
	}

	if err := decodeJSONBody(r.Body, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid login request"})
		return
	}

	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		userID = "demo-user"
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Demo User"
	}

	roles := req.Roles
	if len(roles) == 0 {
		roles = []string{auth.RoleViewer}
	}

	expiresIn := 24 * time.Hour
	if req.ExpiresIn != "" {
		if parsed, err := time.ParseDuration(req.ExpiresIn); err == nil && parsed > 0 {
			expiresIn = parsed
		}
	}

	token, err := s.authManager.GenerateTokenWithRoles(userID, roles, expiresIn)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	user := auth.User{
		ID:    userID,
		Name:  name,
		Email: strings.TrimSpace(req.Email),
		Roles: roles,
		Meta:  map[string]string{"login_source": "demo"},
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":         token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    token.TokenType,
		"expires_at":    token.ExpiresAt,
		"user":          user,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if s.auditor != nil && userID != "" {
		s.auditor.Log(r.Context(), userID, "logout", "/api/v1/auth/logout", "success", map[string]interface{}{
			"remote_addr": r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// Middleware functions.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)

		fields := logrus.Fields{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      recorder.statusCode,
			"duration_ms": time.Since(start).Milliseconds(),
			"remote_addr": r.RemoteAddr,
		}
		if userID := auth.UserIDFromContext(r.Context()); userID != "" {
			fields["user_id"] = userID
		}
		s.logger.WithFields(fields).Info("request completed")
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/login" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.auditAuthFailure(r, "missing authorization header")
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		token, err := parseBearerToken(authHeader)
		if err != nil {
			s.auditAuthFailure(r, err.Error())
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		validated, err := s.authManager.ValidateTokenWithClaims(token)
		if err != nil {
			s.auditAuthFailure(r, err.Error())
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := auth.ContextWithUser(r.Context(), validated.UserID, validated.Roles)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	allowedMethods := "GET, POST, PUT, DELETE, PATCH, OPTIONS"
	allowedHeaders := "Content-Type, Authorization, X-Request-ID, X-CSRF-Token"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && containsOrigin(s.config.CORSOrigins, origin) {
			if len(s.config.CORSOrigins) == 1 && s.config.CORSOrigins[0] == "*" && !s.config.CORSAllowCredentials {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			w.Header().Set("Access-Control-Max-Age", strconv.FormatInt(int64(s.config.CORSMaxAge/time.Second), 10))
			if s.config.CORSAllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) auditAuthFailure(r *http.Request, reason string) {
	if s.auditor == nil {
		return
	}

	s.auditor.Log(r.Context(), "", "authentication", r.URL.Path, "denied", map[string]interface{}{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"reason":      reason,
	})
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func decodeJSONBody(body io.Reader, dest any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func parseBearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid bearer token")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("empty bearer token")
	}

	return token, nil
}
