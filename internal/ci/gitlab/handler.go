package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// APIHandler handles HTTP requests for GitLab CI integration
type APIHandler struct {
	logger         *logrus.Logger
	client         *GitLabClient
	webhookHandler *WebhookHandler
	orchestrator   *OrchestrationController
	allowCORS      bool
	corsOrigins    []string
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(logger *logrus.Logger, client *GitLabClient, webhookHandler *WebhookHandler, orchestrator *OrchestrationController) *APIHandler {
	return &APIHandler{
		logger:         logger,
		client:         client,
		webhookHandler: webhookHandler,
		orchestrator:   orchestrator,
		allowCORS:      true,
		corsOrigins:    []string{"*"},
	}
}

// SetCORS sets CORS configuration
func (h *APIHandler) SetCORS(allow bool, origins []string) {
	h.allowCORS = allow
	h.corsOrigins = origins
}

// middlewareCORS applies CORS headers
func (h *APIHandler) middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		origin := "*"
		if len(h.corsOrigins) > 0 {
			origin = h.corsOrigins[0]
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Gitlab-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// middlewareLogger logs requests
func (h *APIHandler) middlewareLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
		}).Debug("API request received")

		next.ServeHTTP(w, r)
	})
}

// middlewareAuth validates authentication (placeholder)
func (h *APIHandler) middlewareAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for webhook endpoint
		if r.URL.Path == "/api/v1/ci/gitlab/webhook" {
			next.ServeHTTP(w, r)
			return
		}

		// In production, validate API token or session
		// For now, skip authentication in dev mode
		if h.logger.Level >= logrus.DebugLevel {
			h.logger.Debug("Authentication skipped (development mode)")
		}

		next.ServeHTTP(w, r)
	})
}

// ServeHTTP implements http.Handler
func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Apply middleware
	next := http.Handler(http.HandlerFunc(h.handleRequests))
	next = h.middlewareAuth(next)
	next = h.middlewareLogger(next)
	next = h.middlewareCORS(next)
	next.ServeHTTP(w, r)
}

// handleRequests dispatches requests to appropriate handlers
func (h *APIHandler) handleRequests(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/api/v1/ci/gitlab/webhook" && r.Method == "POST":
		h.handleWebhook(w, r)
	case strings.HasPrefix(path, "/api/v1/ci/gitlab/projects"):
		h.handleProjects(w, r)
	case strings.HasPrefix(path, "/api/v1/ci/gitlab/pipelines"):
		h.handlePipelines(w, r)
	case strings.HasPrefix(path, "/api/v1/ci/gitlab/jobs"):
		h.handleJobs(w, r)
	case strings.HasPrefix(path, "/api/v1/ci/gitlab/runners"):
		h.handleRunners(w, r)
	case strings.HasPrefix(path, "/api/v1/ci/gitlab/merge-requests"):
		h.handleMergeRequests(w, r)
	case path == "/api/v1/ci/gitlab/webhook":
		h.handleWebhook(w, r)
	case strings.HasPrefix(path, "/api/v1/ci/gitlab/health"):
		h.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleWebhook handles webhook endpoint
func (h *APIHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.webhookHandler.Handle(w, r)
}

// handleProjects handles project operations
func (h *APIHandler) handleProjects(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ci/gitlab/projects")

	switch path {
	case "", "/":
		h.handleListProjects(w, r)
	case "/health":
		h.handleProjectHealth(w, r)
	default:
		// Try to extract project ID
		parts := strings.Split(path, "/")
		if len(parts) >= 1 {
			if projectID, err := strconv.Atoi(parts[0]); err == nil {
				if len(parts) > 1 && parts[1] == "pipelines" {
					h.handleProjectPipelines(w, r, projectID)
				} else if len(parts) > 1 && parts[1] == "jobs" {
					h.handleProjectJobs(w, r, projectID)
				} else if parts[1] == "health" {
					h.handleProjectHealth(w, r)
				} else {
					h.handleGetProject(w, r, projectID)
				}
				return
			}
		}
		http.NotFound(w, r)
	}
}

// handleGetProject handles get project request
func (h *APIHandler) handleGetProject(w http.ResponseWriter, r *http.Request, projectID int) {
	ctx := r.Context()

	// Fetch from GitLab
	project, err := h.client.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch project")
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, project)
}

// handleListProjects handles list projects request
func (h *APIHandler) handleListProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	params := r.URL.Query()
	opts := &ListProjectsOptions{}

	if page := params.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			opts.Page = p
		}
	}

	if perPage := params.Get("per_page"); perPage != "" {
		if p, err := strconv.Atoi(perPage); err == nil {
			opts.PerPage = p
		}
	}

	if search := params.Get("search"); search != "" {
		opts.Search = search
	}

	if owned := params.Get("owned"); owned == "true" {
		val := true
		opts.Owned = &val
	}

	if membership := params.Get("membership"); membership == "true" {
		val := true
		opts.Membership = &val
	}

	if visibility := params.Get("visibility"); visibility != "" {
		opts.Visibility = visibility
	}

	// Fetch projects
	projects, err := h.client.ListProjects(ctx, opts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list projects")
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	// Return paginated response
	response := map[string]interface{}{
		"projects": projects,
		"total":    len(projects),
		"page":     opts.Page,
		"per_page": opts.PerPage,
	}

	h.writeJSON(w, response)
}

// handleProjectHealth handles project health check
func (h *APIHandler) handleProjectHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract project ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ci/gitlab/projects")
	paths := strings.Split(path, "/")
	if len(paths) >= 2 {
		if projectID, err := strconv.Atoi(paths[0]); err == nil {
			// Get project health
			status, err := h.client.GetProjectStatus(ctx, projectID)
			if err != nil {
				h.logger.WithError(err).Error("Failed to get project status")
				h.writeError(w, err, http.StatusInternalServerError)
				return
			}

			h.writeJSON(w, status)
			return
		}
	}

	http.NotFound(w, r)
}

// handleProjectPipelines handles project pipelines
func (h *APIHandler) handleProjectPipelines(w http.ResponseWriter, r *http.Request, projectID int) {
	ctx := r.Context()

	// Parse query parameters
	params := r.URL.Query()
	opts := &ListPipelinesOptions{}

	if page := params.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			opts.Page = p
		}
	}

	if perPage := params.Get("per_page"); perPage != "" {
		if p, err := strconv.Atoi(perPage); err == nil {
			opts.PerPage = p
		}
	}

	if status := params.Get("status"); status != "" {
		opts.Status = status
	}

	if ref := params.Get("ref"); ref != "" {
		opts.Ref = ref
	}

	if branch := params.Get("branch"); branch != "" {
		opts.Branch = branch
	}

	if sha := params.Get("sha"); sha != "" {
		opts.SHA = sha
	}

	// Fetch pipelines
	pipelines, err := h.client.ListPipelines(ctx, projectID, opts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list pipelines")
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"pipelines": pipelines,
		"total":     len(pipelines),
	}

	h.writeJSON(w, response)
}

// handleProjectJobs handles project jobs
func (h *APIHandler) handleProjectJobs(w http.ResponseWriter, r *http.Request, projectID int) {
	ctx := r.Context()

	// Parse query parameters
	params := r.URL.Query()
	opts := &ListJobsOptions{}

	if page := params.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			opts.Page = p
		}
	}

	if perPage := params.Get("per_page"); perPage != "" {
		if p, err := strconv.Atoi(perPage); err == nil {
			opts.PerPage = p
		}
	}

	if status := params.Get("status"); status != "" {
		opts.Status = status
	}

	if ref := params.Get("ref"); ref != "" {
		opts.Ref = ref
	}

	// Fetch jobs
	jobs, err := h.client.ListJobs(ctx, projectID, opts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list jobs")
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"jobs":  jobs,
		"total": len(jobs),
	}

	h.writeJSON(w, response)
}

// handlePipelines handles pipeline operations
func (h *APIHandler) handlePipelines(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		h.handleListPipelines(w, r, ctx)
	case "POST":
		h.handleCreatePipeline(w, r, ctx)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListPipelines handles list pipelines request
func (h *APIHandler) handleListPipelines(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	projectID, err := h.getProjectIDFromPath(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	// Parse query parameters
	params := r.URL.Query()
	opts := &ListPipelinesOptions{}

	if page := params.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			opts.Page = p
		}
	}

	if perPage := params.Get("per_page"); perPage != "" {
		if p, err := strconv.Atoi(perPage); err == nil {
			opts.PerPage = p
		}
	}

	if status := params.Get("status"); status != "" {
		opts.Status = status
	}

	if ref := params.Get("ref"); ref != "" {
		opts.Ref = ref
	}

	// Fetch pipelines
	pipelines, err := h.client.ListPipelines(ctx, projectID, opts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list pipelines")
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"pipelines": pipelines,
		"total":     len(pipelines),
	})
}

// handleCreatePipeline handles create pipeline request
func (h *APIHandler) handleCreatePipeline(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	projectID, err := h.getProjectIDFromPath(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	// Parse request body
	var body struct {
		Ref     string `json:"ref"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logger.WithError(err).Error("Failed to decode request body")
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	// Create pipeline
	pipeline, err := h.client.CreatePipeline(ctx, projectID, body.Ref, body.Message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create pipeline")
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, pipeline, http.StatusCreated)
}

// handleJobs handles job operations
func (h *APIHandler) handleJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		h.handleListJobs(w, r, ctx)
	case "POST":
		h.handleJobAction(w, r, ctx)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListJobs handles list jobs request
func (h *APIHandler) handleListJobs(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	projectID, err := h.getProjectIDFromPath(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	// Parse query parameters
	params := r.URL.Query()
	opts := &ListJobsOptions{}

	if page := params.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			opts.Page = p
		}
	}

	if perPage := params.Get("per_page"); perPage != "" {
		if p, err := strconv.Atoi(perPage); err == nil {
			opts.PerPage = p
		}
	}

	if status := params.Get("status"); status != "" {
		opts.Status = status
	}

	if ref := params.Get("ref"); ref != "" {
		opts.Ref = ref
	}

	// Fetch jobs
	jobs, err := h.client.ListJobs(ctx, projectID, opts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list jobs")
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"jobs":  jobs,
		"total": len(jobs),
	})
}

// handleJobAction handles job actions (retry, cancel, etc.)
func (h *APIHandler) handleJobAction(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	action := r.URL.Query().Get("action")
	if action == "" {
		http.Error(w, "Missing action parameter", http.StatusBadRequest)
		return
	}

	projectID, err := h.getProjectIDFromPath(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	// Extract job ID
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ci/gitlab/jobs")
	paths := strings.Split(path, "/")
	if len(paths) < 2 {
		http.Error(w, "Missing job ID", http.StatusBadRequest)
		return
	}

	jobID, err := strconv.ParseInt(paths[1], 10, 64)
	if err != nil {
		h.logger.WithError(err).Error("Invalid job ID")
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	switch action {
	case "cancel":
		err = h.client.CancelJob(ctx, projectID, jobID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to cancel job")
			h.writeError(w, err, http.StatusInternalServerError)
			return
		}
	case "retry":
		job, err := h.client.RetryJob(ctx, projectID, jobID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to retry job")
			h.writeError(w, err, http.StatusInternalServerError)
			return
		}
		h.writeJSON(w, job, http.StatusCreated)
		return
	default:
		http.Error(w, fmt.Sprintf("Unknown action: %s", action), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// handleRunners handles runner operations
func (h *APIHandler) handleRunners(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		h.handleListRunners(w, r, ctx)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListRunners handles list runners request
func (h *APIHandler) handleListRunners(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	runners, err := h.client.ListRunners(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list runners")
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"runners": runners,
		"total":   len(runners),
	})
}

// handleMergeRequests handles merge request operations
func (h *APIHandler) handleMergeRequests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		h.handleListMergeRequests(w, r, ctx)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListMergeRequests handles list merge requests request
func (h *APIHandler) handleListMergeRequests(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	projectID, err := h.getProjectIDFromPath(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	// Parse query parameters
	params := r.URL.Query()
	opts := &ListMergeRequestsOptions{}

	if page := params.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			opts.Page = p
		}
	}

	if perPage := params.Get("per_page"); perPage != "" {
		if p, err := strconv.Atoi(perPage); err == nil {
			opts.PerPage = p
		}
	}

	if state := params.Get("state"); state != "" {
		opts.State = state
	}

	if sourceBranch := params.Get("source_branch"); sourceBranch != "" {
		opts.SourceBranch = sourceBranch
	}

	if targetBranch := params.Get("target_branch"); targetBranch != "" {
		opts.TargetBranch = targetBranch
	}

	// Fetch merge requests
	mrs, err := h.client.ListMergeRequests(ctx, projectID, opts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list merge requests")
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"merge_requests": mrs,
		"total":          len(mrs),
	})
}

// handleHealth handles health check requests
func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := h.client.GetHealthStatus(r.Context())
	h.writeJSON(w, status)
}

// writeJSON writes JSON response
func (h *APIHandler) writeJSON(w http.ResponseWriter, data interface{}, status ...int) {
	w.Header().Set("Content-Type", "application/json")
	if len(status) > 0 {
		w.WriteHeader(status[0])
	} else {
		w.WriteHeader(http.StatusOK)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeError writes error response
func (h *APIHandler) writeError(w http.ResponseWriter, err error, status int) {
	h.logger.WithError(err).Error("API error")

	response := map[string]string{
		"error": err.Error(),
	}
	h.writeJSON(w, response, status)
}

// getProjectIDFromPath extracts project ID from URL path
func (h *APIHandler) getProjectIDFromPath(r *http.Request) (int, error) {
	path := r.URL.Path

	// Remove base path
	path = strings.TrimPrefix(path, "/api/v1/ci/gitlab/projects")
	path = strings.TrimPrefix(path, "/api/v1/ci/gitlab/jobs")
	path = strings.TrimPrefix(path, "/api/v1/ci/gitlab/pipelines")

	paths := strings.Split(path, "/")
	if len(paths) > 0 && paths[0] != "" {
		projectID, err := strconv.Atoi(paths[0])
		if err != nil {
			return 0, fmt.Errorf("invalid project ID: %s", paths[0])
		}
		return projectID, nil
	}

	return 0, fmt.Errorf("project ID not found in path")
}

// RegisterRoutes registers API routes
func (h *APIHandler) RegisterRoutes(router *mux.Router) {
	// Apply middleware to all routes
	router.Use(h.middlewareCORS)
	router.Use(h.middlewareLogger)
	router.Use(h.middlewareAuth)

	// Webhook endpoint
	router.HandleFunc("/api/v1/ci/gitlab/webhook", h.handleWebhook).Methods("POST")

	// Projects endpoints
	router.HandleFunc("/api/v1/ci/gitlab/projects", h.handleListProjects).Methods("GET")
	router.HandleFunc("/api/v1/ci/gitlab/projects/{id:[0-9]+}", h.handleGetProjectWrapper).Methods("GET")
	router.HandleFunc("/api/v1/ci/gitlab/projects/{id:[0-9]+}/health", h.handleProjectHealthWrapper).Methods("GET")
	router.HandleFunc("/api/v1/ci/gitlab/projects/{id:[0-9]+}/pipelines", h.handleProjectPipelinesWrapper).Methods("GET")
	router.HandleFunc("/api/v1/ci/gitlab/projects/{id:[0-9]+}/jobs", h.handleProjectJobsWrapper).Methods("GET")

	// Pipelines endpoints
	router.HandleFunc("/api/v1/ci/gitlab/pipelines", h.handlePipelines).Methods("GET", "POST")
	router.HandleFunc("/api/v1/ci/gitlab/pipelines/{id:[0-9]+}", h.handlePipelines).Methods("GET")

	// Jobs endpoints
	router.HandleFunc("/api/v1/ci/gitlab/jobs", h.handleJobs).Methods("GET", "POST")
	router.HandleFunc("/api/v1/ci/gitlab/jobs/{id:[0-9]+}/action/{action}", h.handleJobActionWrapper).Methods("POST")

	// Runners endpoints
	router.HandleFunc("/api/v1/ci/gitlab/runners", h.handleRunners).Methods("GET")

	// Merge requests endpoints
	router.HandleFunc("/api/v1/ci/gitlab/merge-requests", h.handleMergeRequests).Methods("GET")
	router.HandleFunc("/api/v1/ci/gitlab/merge-requests/{id:[0-9]+}", h.handleMergeRequests).Methods("GET")

	// Health check endpoint
	router.HandleFunc("/api/v1/ci/gitlab/health", h.handleHealth).Methods("GET")
}

// Wrapper functions for handlers with additional parameters
func (h *APIHandler) handleGetProjectWrapper(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if projectID, err := strconv.Atoi(vars["id"]); err == nil {
		h.handleGetProject(w, r, projectID)
	} else {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
	}
}

func (h *APIHandler) handleProjectHealthWrapper(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from path for validation
	if strings.HasPrefix(r.URL.Path, "/projects/") {
		h.handleProjectHealth(w, r)
	} else {
		http.Error(w, "Invalid project path", http.StatusBadRequest)
	}
}

func (h *APIHandler) handleProjectPipelinesWrapper(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if projectID, err := strconv.Atoi(vars["id"]); err == nil {
		h.handleProjectPipelines(w, r, projectID)
	} else {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
	}
}

func (h *APIHandler) handleProjectJobsWrapper(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if projectID, err := strconv.Atoi(vars["id"]); err == nil {
		h.handleProjectJobs(w, r, projectID)
	} else {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
	}
}

func (h *APIHandler) handleJobActionWrapper(w http.ResponseWriter, r *http.Request) {
	h.handleJobAction(w, r, r.Context())
}
