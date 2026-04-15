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
	metrics     *Metrics
	authManager *auth.Manager
	rbac        *auth.RBAC
	auditor     *Auditor
	rateLimiter *RateLimiter
	aiBackend   aiBackend
	deployer    deploymentManager
	gitops      gitOpsManager
	stateStore  runtimeStateStore
	jobs        *asyncJobManager
	startedAt   time.Time
}

// New creates a new server instance.
func New(cfg *config.Config, logger *logrus.Logger) (*Server, error) {
	metrics := newMetrics()
	stateStore, err := newRuntimeStateStore(cfg)
	if err != nil && cfg != nil && cfg.Environment == "production" {
		return nil, err
	}
	if err != nil && logger != nil {
		logger.WithError(err).Warn("failed to initialize SQL runtime state store; falling back to in-memory state")
	}

	s := &Server{
		config:      cfg,
		logger:      logger,
		router:      mux.NewRouter(),
		metrics:     metrics,
		authManager: auth.NewManager(cfg.SessionSecret),
		rbac:        auth.NewRBAC(),
		auditor:     NewAuditor(),
		rateLimiter: NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow),
		aiBackend:   newAIBackend(cfg, logger),
		deployer:    newDeploymentManager(cfg, logger),
		gitops:      newGitOpsOrchestrator(cfg, logger),
		stateStore:  stateStore,
		jobs:        newAsyncJobManager(cfg, logger),
		startedAt:   time.Now().UTC(),
	}

	s.auditor.SetMetrics(metrics)
	s.auditor.SetStore(stateStore)
	s.auditor.SetLogger(logger)
	s.rateLimiter.SetMetrics(metrics)
	s.rateLimiter.SetStore(stateStore)
	s.rateLimiter.SetLogger(logger)
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
	s.router.Use(s.metrics.Middleware)
	s.router.Use(s.rateLimiter.Middleware)

	s.router.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)
	s.router.HandleFunc("/live", s.handleLive).Methods(http.MethodGet)
	s.router.HandleFunc("/ready", s.handleReady).Methods(http.MethodGet)
	s.router.HandleFunc("/metrics", s.handleMetrics).Methods(http.MethodGet)

	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.authMiddleware)
	api.Use(s.auditor.Middleware)

	api.Handle("/catalog/services", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleListServices))).Methods(http.MethodGet)
	api.Handle("/catalog/search", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleSearchCatalog))).Methods(http.MethodGet)
	api.Handle("/catalog/overview", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleCatalogOverview))).Methods(http.MethodGet)
	api.Handle("/catalog/services/{id}", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleServiceInsight))).Methods(http.MethodGet)
	api.Handle("/catalog/services/{id}/analysis", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleServiceInsight))).Methods(http.MethodGet)
	api.Handle("/platform/status", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handlePlatformStatus))).Methods(http.MethodGet)
	api.Handle("/platform/observability", s.rbac.Middleware("catalog", "read")(http.HandlerFunc(s.handleObservability))).Methods(http.MethodGet)
	api.Handle("/jobs", s.rbac.Middleware("services", "read")(http.HandlerFunc(s.handleJobs))).Methods(http.MethodGet)
	api.Handle("/jobs/{id}", s.rbac.Middleware("services", "read")(http.HandlerFunc(s.handleJobStatus))).Methods(http.MethodGet)
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
	if err := s.http.Shutdown(ctx); err != nil {
		return err
	}
	if s.jobs != nil {
		if err := s.jobs.Shutdown(ctx); err != nil {
			return err
		}
	}
	if s.stateStore != nil {
		_ = s.stateStore.Close()
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := s.buildPlatformStatus()
	if s.metrics != nil {
		s.metrics.UpdatePlatformSnapshot(status)
		s.metrics.UpdateOperationalSnapshots(status.Audit, status.RateLimiting)
	}
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
	if readyStatus == statusUnready {
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
	status := s.buildPlatformStatus()
	if s.metrics != nil {
		s.metrics.UpdatePlatformSnapshot(status)
		s.metrics.UpdateOperationalSnapshots(status.Audit, status.RateLimiting)
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if s.metrics == nil {
		http.Error(w, "metrics unavailable", http.StatusServiceUnavailable)
		return
	}

	s.metrics.Handler().ServeHTTP(w, r)
}

func (s *Server) handleServiceInsight(w http.ResponseWriter, r *http.Request) {
	serviceID := strings.TrimSpace(mux.Vars(r)["id"])
	portfolio := buildPortfolioIntelligence(demoCatalog)
	for _, service := range demoCatalog {
		if service.ID == serviceID {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"service":   buildCatalogView(service),
				"insight":   buildServiceInsight(service),
				"portfolio": portfolio,
				"overview":  catalogSummary(demoCatalog),
				"brief":     buildReleaseBrief(service, portfolio),
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
	queryResult := buildQueryResult(payload.Query, demoCatalog, payload.ServiceID)
	ctx, cancel := newAIRequestContext(r.Context(), s.config.AITimeout)
	defer cancel()

	if req, ok := parseArgoCDDeploymentPrompt(payload.Query, s.config.KubernetesNamespace); ok {
		if !s.rbac.CanAccessRoles(roles, "services", "deploy") {
			s.writeError(w, http.StatusForbidden, "deployment access denied")
			return
		}

		operationStart := time.Now()
		job, record, err := s.enqueueDeploymentJob(ctx, req, payload.Query, userID, roles, "gitops-argocd", "deployment_apply_argocd", "ai")
		if s.metrics != nil {
			outcome := "success"
			if err != nil {
				outcome = "error"
			}
			s.metrics.RecordDeploymentRequest("argocd", "apply", time.Since(operationStart), outcome)
			s.metrics.RecordAIRequest("gitops-argocd", "deployment_apply_argocd", time.Since(operationStart), outcome)
		}
		if err != nil {
			s.writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"response":          fmt.Sprintf("Argo CD deployment request for %s was queued for async execution.", record.Name),
			"backend":           "gitops-argocd",
			"query":             payload.Query,
			"user_id":           userID,
			"roles":             roles,
			"intent":            "deployment_apply_argocd",
			"deployment":        record,
			"job":               job,
			"job_status_url":    "/api/v1/jobs/" + job.ID,
			"execution_plan":    record.ExecutionPlan,
			"execution_state":   record.ExecutionState,
			"action_plan":       buildActionPlan(withIntent(queryResult, "deployment_apply_argocd"), record, nil),
			"generated_text":    fmt.Sprintf("Created GitHub delivery branch %s and queued Argo CD application %s.", fallbackString(record.Revision, "generated"), fallbackString(record.ApplicationName, record.Name)),
			"delivery_provider": "argocd",
		})
		return
	}

	if namespace, name, ok := parseArgoCDStatusPrompt(payload.Query, s.config.KubernetesNamespace); ok {
		if !s.rbac.CanAccessRoles(roles, "services", "read") {
			s.writeError(w, http.StatusForbidden, "deployment status access denied")
			return
		}

		operationStart := time.Now()
		record, err := s.gitops.ArgoCDDeploymentStatus(ctx, namespace, name)
		if s.metrics != nil {
			outcome := "success"
			if err != nil {
				outcome = "error"
			}
			s.metrics.RecordDeploymentRequest("argocd", "status", time.Since(operationStart), outcome)
			s.metrics.RecordAIRequest("gitops-argocd", "deployment_status_argocd", time.Since(operationStart), outcome)
		}
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
			"execution_plan":    record.ExecutionPlan,
			"execution_state":   record.ExecutionState,
			"action_plan":       buildActionPlan(withIntent(queryResult, "deployment_status_argocd"), record, nil),
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

		operationStart := time.Now()
		job, record, err := s.enqueueDeploymentJob(ctx, req, payload.Query, userID, roles, "deployment-control-plane", "deployment_apply", "ai")
		if s.metrics != nil {
			outcome := "success"
			if err != nil {
				outcome = "error"
			}
			s.metrics.RecordDeploymentRequest("kubernetes", "apply", time.Since(operationStart), outcome)
			s.metrics.RecordAIRequest("deployment-control-plane", "deployment_apply", time.Since(operationStart), outcome)
		}
		if err != nil {
			s.writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"response":        fmt.Sprintf("Deployment %s was queued for async execution in namespace %s.", record.Name, record.Namespace),
			"backend":         "deployment-control-plane",
			"query":           payload.Query,
			"user_id":         userID,
			"roles":           roles,
			"intent":          "deployment_apply",
			"deployment":      record,
			"job":             job,
			"job_status_url":  "/api/v1/jobs/" + job.ID,
			"execution_plan":  record.ExecutionPlan,
			"execution_state": record.ExecutionState,
			"action_plan":     buildActionPlan(catalogQueryResult{Intent: "deployment_apply"}, record, nil),
			"generated_text":  fmt.Sprintf("Queued deployment %s using image %s and service type %s.", record.Name, record.Image, record.ServiceType),
		})
		return
	}

	if namespace, name, ok := parseDeploymentStatusPrompt(payload.Query, s.config.KubernetesNamespace); ok {
		if !s.rbac.CanAccessRoles(roles, "services", "read") {
			s.writeError(w, http.StatusForbidden, "deployment status access denied")
			return
		}

		operationStart := time.Now()
		record, err := s.deployer.Status(ctx, namespace, name)
		if s.metrics != nil {
			outcome := "success"
			if err != nil {
				outcome = "error"
			}
			s.metrics.RecordDeploymentRequest("kubernetes", "status", time.Since(operationStart), outcome)
			s.metrics.RecordAIRequest("deployment-control-plane", "deployment_status", time.Since(operationStart), outcome)
		}
		if err != nil {
			s.writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"response":        fmt.Sprintf("Deployment %s is %s in namespace %s with %d/%d ready replicas.", record.Name, record.Phase, record.Namespace, record.ReadyReplicas, record.Replicas),
			"backend":         "deployment-control-plane",
			"query":           payload.Query,
			"user_id":         userID,
			"roles":           roles,
			"intent":          "deployment_status",
			"deployment":      record,
			"execution_plan":  record.ExecutionPlan,
			"execution_state": record.ExecutionState,
			"action_plan":     buildActionPlan(withIntent(queryResult, "deployment_status"), record, nil),
			"generated_text":  record.Message,
		})
		return
	}

	if req, ok := parseInfrastructurePrompt(payload.Query); ok {
		if !s.rbac.CanAccessRoles(roles, "services", "deploy") {
			s.writeError(w, http.StatusForbidden, "infrastructure access denied")
			return
		}

		operationStart := time.Now()
		job, record, err := s.enqueueInfrastructureJob(ctx, req, payload.Query, userID, roles, "gitops-infrastructure", "infrastructure_apply_"+req.Provider, "ai")
		if s.metrics != nil {
			outcome := "success"
			if err != nil {
				outcome = "error"
			}
			s.metrics.RecordDeploymentRequest(req.Provider, "infrastructure_apply", time.Since(operationStart), outcome)
			s.metrics.RecordAIRequest("gitops-infrastructure", "infrastructure_apply_"+req.Provider, time.Since(operationStart), outcome)
		}
		if err != nil {
			s.writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"response":             record.Message,
			"backend":              "gitops-infrastructure",
			"query":                payload.Query,
			"user_id":              userID,
			"roles":                roles,
			"intent":               "infrastructure_apply_" + record.Provider,
			"infrastructure":       record,
			"job":                  job,
			"job_status_url":       "/api/v1/jobs/" + job.ID,
			"execution_plan":       record.ExecutionPlan,
			"execution_state":      record.ExecutionState,
			"action_plan":          buildActionPlan(withIntent(queryResult, "infrastructure_apply_"+record.Provider), nil, record),
			"generated_text":       record.Message,
			"infrastructure_stack": record.Provider,
		})
		return
	}

	aiStart := time.Now()
	answer, source := queryAI(ctx, s.aiBackend, aiQueryRequest{
		Query:     payload.Query,
		Services:  demoCatalog,
		Focus:     queryResult.FocusService,
		Portfolio: queryResult.Portfolio,
		Intent:    queryResult.Intent,
		UserID:    userID,
		Roles:     roles,
	}, s.logger)
	if s.metrics != nil {
		outcome := "success"
		if strings.TrimSpace(answer) == "" {
			outcome = "error"
		}
		s.metrics.RecordAIRequest(source, queryResult.Intent, time.Since(aiStart), outcome)
	}

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
		"action_plan":       buildActionPlan(queryResult, nil, nil),
	})
}

func (s *Server) handleApplyDeployment(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req deploymentApplyRequest
	if err := decodeJSONBody(r.Body, &req); err != nil && !errors.Is(err, io.EOF) {
		s.writeError(w, http.StatusBadRequest, "invalid deployment request")
		return
	}

	job, record, err := s.enqueueDeploymentJob(r.Context(), req, fmt.Sprintf("%s deployment request queued through the control plane", req.Name), auth.UserIDFromContext(r.Context()), auth.RolesFromContext(r.Context()), "deployment-control-plane", "deployment_apply", "api")
	if s.metrics != nil {
		outcome := "success"
		if err != nil {
			outcome = "error"
		}
		s.metrics.RecordDeploymentRequest("kubernetes", "apply", time.Since(start), outcome)
	}
	if err != nil {
		s.writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"response":        fmt.Sprintf("Deployment %s was queued for async execution in namespace %s.", record.Name, record.Namespace),
		"backend":         "deployment-control-plane",
		"intent":          "deployment_apply",
		"deployment":      record,
		"job":             job,
		"job_status_url":  "/api/v1/jobs/" + job.ID,
		"execution_plan":  record.ExecutionPlan,
		"execution_state": record.ExecutionState,
		"action_plan":     buildActionPlan(catalogQueryResult{Intent: "deployment_apply"}, record, nil),
	})
}

func (s *Server) handleDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	record, err := s.deployer.Status(r.Context(), vars["namespace"], vars["name"])
	if s.metrics != nil {
		outcome := "success"
		if err != nil {
			outcome = "error"
		}
		s.metrics.RecordDeploymentRequest("kubernetes", "status", time.Since(start), outcome)
	}
	if err != nil {
		s.writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployment": record,
	})
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if s.jobs == nil {
		s.writeError(w, http.StatusServiceUnavailable, "job manager unavailable")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  s.jobs.List(),
		"stats": s.jobs.Stats(),
	})
}

func (s *Server) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	if s.jobs == nil {
		s.writeError(w, http.StatusServiceUnavailable, "job manager unavailable")
		return
	}

	job, ok := s.jobs.Get(strings.TrimSpace(mux.Vars(r)["id"]))
	if !ok {
		s.writeError(w, http.StatusNotFound, "job not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"job": job,
	})
}

func (s *Server) enqueueDeploymentJob(
	ctx context.Context,
	req deploymentApplyRequest,
	query string,
	userID string,
	roles []string,
	backend string,
	intent string,
	source string,
) (*asyncJob, *deploymentRecord, error) {
	if s.jobs == nil {
		return nil, nil, errors.New("job manager unavailable")
	}

	normalizedReq, err := normalizeDeploymentRequest(req, s.config.KubernetesNamespace)
	if err != nil {
		return nil, nil, err
	}

	plannedRecord := queuedDeploymentRecord(normalizedReq, intent)
	submission := asyncJobSubmission{
		Kind:          asyncJobKindDeploymentApply,
		Intent:        intent,
		Backend:       backend,
		Source:        source,
		UserID:        userID,
		Roles:         roles,
		Summary:       fmt.Sprintf("Deployment request for %s queued for async execution.", normalizedReq.Name),
		Detail:        query,
		ResourceType:  "deployment",
		ResourceName:  normalizedReq.Name,
		Namespace:     normalizedReq.Namespace,
		Provider:      normalizedReq.Delivery,
		Route:         plannedRecord.ExecutionPlan.Route,
		Mode:          plannedRecord.ExecutionPlan.Mode,
		ExecutionPlan: plannedRecord.ExecutionPlan,
		Task: func(jobCtx context.Context) (*asyncJobResult, error) {
			var record *deploymentRecord
			var err error
			if normalizedReq.Delivery == "argocd" {
				record, err = s.gitops.ApplyArgoCDDeployment(jobCtx, normalizedReq)
			} else {
				record, err = s.deployer.Apply(jobCtx, normalizedReq)
			}
			if err != nil {
				return nil, err
			}
			return &asyncJobResult{Deployment: record}, nil
		},
	}

	job, err := s.jobs.Submit(ctx, submission)
	if err != nil {
		return nil, nil, err
	}
	plannedRecord.JobID = job.ID
	plannedRecord.JobStatus = job.Status
	plannedRecord.ExecutionState = job.Status
	plannedRecord.Message = fmt.Sprintf("Deployment request for %s was accepted and queued as %s.", plannedRecord.Name, job.ID)
	plannedRecord.ExecutionPlan = cloneExecutionPlan(job.ExecutionPlan)
	return job, plannedRecord, nil
}

func (s *Server) enqueueInfrastructureJob(
	ctx context.Context,
	req infrastructureApplyRequest,
	query string,
	userID string,
	roles []string,
	backend string,
	intent string,
	source string,
) (*asyncJob, *infrastructureRecord, error) {
	if s.jobs == nil {
		return nil, nil, errors.New("job manager unavailable")
	}

	normalizedReq, err := normalizeInfrastructureRequest(req)
	if err != nil {
		return nil, nil, err
	}

	plannedRecord := queuedInfrastructureRecord(normalizedReq, intent)
	submission := asyncJobSubmission{
		Kind:          asyncJobKindInfrastructureApply,
		Intent:        intent,
		Backend:       backend,
		Source:        source,
		UserID:        userID,
		Roles:         roles,
		Summary:       fmt.Sprintf("%s infrastructure request queued for async execution.", strings.Title(normalizedReq.Provider)),
		Detail:        query,
		ResourceType:  "infrastructure",
		ResourceName:  normalizedReq.Name,
		Namespace:     normalizedReq.TargetNamespace,
		Provider:      normalizedReq.Provider,
		Route:         plannedRecord.ExecutionPlan.Route,
		Mode:          plannedRecord.ExecutionPlan.Mode,
		ExecutionPlan: plannedRecord.ExecutionPlan,
		Task: func(jobCtx context.Context) (*asyncJobResult, error) {
			record, err := s.gitops.ApplyInfrastructure(jobCtx, normalizedReq)
			if err != nil {
				return nil, err
			}
			return &asyncJobResult{Infrastructure: record}, nil
		},
	}

	job, err := s.jobs.Submit(ctx, submission)
	if err != nil {
		return nil, nil, err
	}
	plannedRecord.JobID = job.ID
	plannedRecord.JobStatus = job.Status
	plannedRecord.ExecutionState = job.Status
	plannedRecord.Message = fmt.Sprintf("%s infrastructure request was accepted and queued as %s.", strings.Title(plannedRecord.Provider), job.ID)
	plannedRecord.ExecutionPlan = cloneExecutionPlan(job.ExecutionPlan)
	return job, plannedRecord, nil
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
