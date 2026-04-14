package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	deployer    deploymentManager
	gitops      gitOpsManager
	startedAt   time.Time
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
		deployer:    newDeploymentManager(cfg, logger),
		gitops:      newGitOpsOrchestrator(cfg, logger),
		startedAt:   time.Now().UTC(),
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
	s.router.HandleFunc("/live", s.handleLive).Methods(http.MethodGet)
	s.router.HandleFunc("/ready", s.handleReady).Methods(http.MethodGet)

	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.authMiddleware)
	api.Use(s.auditor.Middleware)

	api.Handle("/catalog/services", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleListServices))).Methods(http.MethodGet)
	api.Handle("/catalog/search", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleSearchCatalog))).Methods(http.MethodGet)
	api.Handle("/catalog/overview", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleCatalogOverview))).Methods(http.MethodGet)
	api.Handle("/catalog/services/{id}", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleServiceInsight))).Methods(http.MethodGet)
	api.Handle("/catalog/services/{id}/analysis", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleServiceInsight))).Methods(http.MethodGet)
	api.Handle("/platform/status", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handlePlatformStatus))).Methods(http.MethodGet)
	api.Handle("/deployments/applications", s.rbac.Middleware("services", "deploy")(http.HandlerFunc(s.handleApplyDeployment))).Methods(http.MethodPost)
	api.Handle("/deployments/applications/{namespace}/{name}", s.rbac.Middleware("services", "read")(http.HandlerFunc(s.handleDeploymentStatus))).Methods(http.MethodGet)

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
	status := s.buildPlatformStatus()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     status.Status,
		"started_at": status.StartedAt,
		"uptime":     status.Uptime,
		"checks":     status.Checks,
	})
}

func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	readyStatus, checks := s.buildReadinessStatus()
	httpStatus := http.StatusOK
	if readyStatus != statusReady {
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, map[string]interface{}{
		"status": readyStatus,
		"checks": checks,
	})
}

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"services":  buildCatalogViews(demoCatalog),
		"overview":  catalogSummary(demoCatalog),
		"portfolio": buildPortfolioIntelligence(demoCatalog),
	})
}

func (s *Server) handleSearchCatalog(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("query"))
	}

	results := make([]catalogServiceView, 0, len(demoCatalog))
	for _, service := range demoCatalog {
		searchable := strings.ToLower(strings.Join(append([]string{
			service.Name,
			service.Description,
			service.Owner,
			service.Team,
			service.Tier,
		}, service.Dependencies...), " "))
		if query == "" || strings.Contains(searchable, strings.ToLower(query)) {
			results = append(results, buildCatalogView(service))
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"query":     query,
		"results":   results,
		"overview":  catalogSummary(demoCatalog),
		"portfolio": buildPortfolioIntelligence(demoCatalog),
	})
}

func (s *Server) handleCatalogOverview(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"overview":  catalogSummary(demoCatalog),
		"portfolio": buildPortfolioIntelligence(demoCatalog),
		"services":  buildCatalogViews(demoCatalog),
	})
}

func (s *Server) handlePlatformStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.buildPlatformStatus())
}

func (s *Server) handleServiceInsight(w http.ResponseWriter, r *http.Request) {
	serviceID := strings.TrimSpace(mux.Vars(r)["id"])
	for _, service := range demoCatalog {
		if service.ID == serviceID {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"service":   buildCatalogView(service),
				"insight":   buildServiceInsight(service),
				"portfolio": buildPortfolioIntelligence(demoCatalog),
				"overview":  catalogSummary(demoCatalog),
				"analysis":  buildQueryResult(service.Name, demoCatalog, service.ID),
			})
			return
		}
	}

	s.writeError(w, http.StatusNotFound, "service not found")
}

func (s *Server) handleAIQuery(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Query     string `json:"query"`
		Text      string `json:"text"`
		ServiceID string `json:"service_id"`
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

	if req, ok := parseArgoCDDeploymentPrompt(payload.Query, s.config.KubernetesNamespace); ok {
		if !s.rbac.CanAccessRoles(roles, "services", "deploy") {
			s.writeError(w, http.StatusForbidden, "deployment access denied")
			return
		}

		record, err := s.gitops.ApplyArgoCDDeployment(ctx, req)
		if err != nil {
			s.writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"response":          fmt.Sprintf("Argo CD deployed %s from GitHub branch %s and the rollout is %s in namespace %s.", record.Name, record.Revision, record.Phase, record.Namespace),
			"backend":           "gitops-argocd",
			"query":             payload.Query,
			"user_id":           userID,
			"roles":             roles,
			"intent":            "deployment_apply_argocd",
			"deployment":        record,
			"generated_text":    fmt.Sprintf("Created GitHub delivery branch %s and Argo CD application %s.", record.Revision, record.ApplicationName),
			"delivery_provider": "argocd",
		})
		return
	}

	if namespace, name, ok := parseArgoCDStatusPrompt(payload.Query, s.config.KubernetesNamespace); ok {
		if !s.rbac.CanAccessRoles(roles, "services", "read") {
			s.writeError(w, http.StatusForbidden, "deployment status access denied")
			return
		}

		record, err := s.gitops.ArgoCDDeploymentStatus(ctx, namespace, name)
		if err != nil {
			s.writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"response":          fmt.Sprintf("Argo CD reports %s as %s with sync %s and health %s.", record.Name, record.Phase, record.SyncStatus, record.HealthStatus),
			"backend":           "gitops-argocd",
			"query":             payload.Query,
			"user_id":           userID,
			"roles":             roles,
			"intent":            "deployment_status_argocd",
			"deployment":        record,
			"generated_text":    record.Message,
			"delivery_provider": "argocd",
		})
		return
	}

	if req, ok := parseDeploymentPrompt(payload.Query, s.config.KubernetesNamespace); ok {
		if !s.rbac.CanAccessRoles(roles, "services", "deploy") {
			s.writeError(w, http.StatusForbidden, "deployment access denied")
			return
		}

		record, err := s.deployer.Apply(ctx, req)
		if err != nil {
			s.writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"response":       fmt.Sprintf("Deployment %s is %s in namespace %s with %d/%d ready replicas.", record.Name, record.Phase, record.Namespace, record.ReadyReplicas, record.Replicas),
			"backend":        "deployment-control-plane",
			"query":          payload.Query,
			"user_id":        userID,
			"roles":          roles,
			"intent":         "deployment_apply",
			"deployment":     record,
			"generated_text": fmt.Sprintf("Applied deployment %s using image %s and service type %s.", record.Name, record.Image, record.ServiceType),
		})
		return
	}

	if namespace, name, ok := parseDeploymentStatusPrompt(payload.Query, s.config.KubernetesNamespace); ok {
		if !s.rbac.CanAccessRoles(roles, "services", "read") {
			s.writeError(w, http.StatusForbidden, "deployment status access denied")
			return
		}

		record, err := s.deployer.Status(ctx, namespace, name)
		if err != nil {
			s.writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"response":       fmt.Sprintf("Deployment %s is %s in namespace %s with %d/%d ready replicas.", record.Name, record.Phase, record.Namespace, record.ReadyReplicas, record.Replicas),
			"backend":        "deployment-control-plane",
			"query":          payload.Query,
			"user_id":        userID,
			"roles":          roles,
			"intent":         "deployment_status",
			"deployment":     record,
			"generated_text": record.Message,
		})
		return
	}

	if req, ok := parseInfrastructurePrompt(payload.Query); ok {
		if !s.rbac.CanAccessRoles(roles, "services", "deploy") {
			s.writeError(w, http.StatusForbidden, "infrastructure access denied")
			return
		}

		record, err := s.gitops.ApplyInfrastructure(ctx, req)
		if err != nil {
			s.writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"response":             record.Message,
			"backend":              "gitops-infrastructure",
			"query":                payload.Query,
			"user_id":              userID,
			"roles":                roles,
			"intent":               "infrastructure_apply_" + record.Provider,
			"infrastructure":       record,
			"generated_text":       record.Message,
			"infrastructure_stack": record.Provider,
		})
		return
	}

	queryResult := buildQueryResult(payload.Query, demoCatalog, payload.ServiceID)
	answer, source := queryAI(ctx, s.aiBackend, aiQueryRequest{
		Query:     payload.Query,
		Services:  demoCatalog,
		Focus:     queryResult.FocusService,
		Portfolio: queryResult.Portfolio,
		Intent:    queryResult.Intent,
		UserID:    userID,
		Roles:     roles,
	}, s.logger)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"response":          answer,
		"backend":           source,
		"query":             payload.Query,
		"user_id":           userID,
		"roles":             roles,
		"intent":            queryResult.Intent,
		"focus_service":     queryResult.FocusService,
		"portfolio":         queryResult.Portfolio,
		"matching_services": queryResult.MatchingServices,
		"release_readiness": queryResult.ReleaseReadiness,
		"evidence_pack":     queryResult.EvidencePack,
		"ownership_drift":   queryResult.OwnershipDrift,
		"next_steps":        queryResult.NextSteps,
		"key_findings":      queryResult.KeyFindings,
		"generated_text":    answer,
	})
}

func (s *Server) handleApplyDeployment(w http.ResponseWriter, r *http.Request) {
	var req deploymentApplyRequest
	if err := decodeJSONBody(r.Body, &req); err != nil && !errors.Is(err, io.EOF) {
		s.writeError(w, http.StatusBadRequest, "invalid deployment request")
		return
	}

	record, err := s.deployer.Apply(r.Context(), req)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployment": record,
	})
}

func (s *Server) handleDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	record, err := s.deployer.Status(r.Context(), vars["namespace"], vars["name"])
	if err != nil {
		s.writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployment": record,
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
