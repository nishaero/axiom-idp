package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Router handles AI-related HTTP requests
type Router struct {
	logger            *logrus.Logger
	config            *AIConfig
	aiEngine          *RecommendationEngine
	pgVector          *PGVectorEmbeddings
	openAIClient      *OpenAIClient
	promptEngine      PromptEngine
	httpClient        *http.Client
	db                *sql.DB
	mu                sync.RWMutex
}

// QueryRequest represents an AI query request
type QueryRequest struct {
	Text        string            `json:"text"`
	ContextSize int               `json:"context_size"`
	Tools       []string          `json:"tools"`
	UserID      string            `json:"user_id,omitempty"`
	Filter      map[string]string `json:"filter,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
}

// QueryResponse represents an AI query response
type QueryResponse struct {
	ProcessedQuery string                 `json:"processed_query"`
	Answer         string                 `json:"answer"`
	Sources        []string               `json:"sources"`
	Metadata       map[string]interface{} `json:"metadata"`
	TokensUsed     int                    `json:"tokens_used"`
	ProcessingTime string                 `json:"processing_time"`
}

// RecommendationsRequest represents a recommendations request
type RecommendationsRequest struct {
	UserID string   `json:"user_id"`
	Query  string   `json:"query"`
	Limit  int      `json:"limit,omitempty"`
	Filters map[string]interface{} `json:"filters,omitempty"`
}

// RecommendationsResponse represents recommendations response
type RecommendationsResponse struct {
	Recommendations []Recommendation `json:"recommendations"`
	TotalFound    int              `json:"total_found"`
	QueryTime     string           `json:"query_time"`
	UserID        string           `json:"user_id"`
}

// HealthStatus represents the health status of AI services
type HealthStatus struct {
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	Services     map[string]string `json:"services"`
	Metrics      map[string]interface{} `json:"metrics"`
}

// NewRouter creates a new AI router
func NewRouter(logger *logrus.Logger, config *AIConfig, db *sql.DB) *Router {
	return &Router{
		logger:   logger.WithField("component", "ai_router"),
		config:   config,
		db:       db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Init initializes the AI router and all components
func (r *Router) Init(ctx context.Context) error {
	r.logger.Info("Initializing AI router...")

	// Initialize OpenAI client if enabled
	if r.config.OpenAI.Enabled {
		openAIClient := NewOpenAIClient(r.logger, r.config.GetOpenAIClientConfig())
		r.openAIClient = openAIClient
		r.logger.Info("OpenAI client initialized")
	} else {
		r.openAIClient = NewMockOpenAIClient()
		r.logger.Info("Mock OpenAI client initialized (using mock)")
	}

	// Initialize PGVector if enabled
	if r.config.PGVector.Enabled && r.db != nil {
		r.pgVector = NewPGVectorEmbeddings(r.logger, r.db, r.config.GetPGVectorConfig())
		if err := r.pgVector.Initialize(ctx); err != nil {
			r.logger.WithError(err).Warn("Failed to initialize PGVector, using fallback")
			r.pgVector = nil
		} else {
			r.logger.Info("PGVector initialized successfully")
		}
	}

	// Initialize prompt engine
	r.promptEngine = NewDefaultPromptEngine()

	// Initialize recommendation engine
	r.aiEngine = NewRecommendationEngine(r.logger, r.openAIClient, r.pgVector)

	r.logger.Info("AI router initialized successfully")
	return nil
}

// SetupRoutes configures all AI-related routes
func (r *Router) SetupRoutes(api *mux.Router) {
	// AI query endpoint
	api.HandleFunc("/ai/query", r.handleAIQuery).Methods(http.MethodPost)

	// AI recommendations endpoint
	api.HandleFunc("/ai/recommendations", r.handleRecommendations).Methods(http.MethodPost)

	// AI health check endpoint
	api.HandleFunc("/ai/health", r.handleHealth).Methods(http.MethodGet)

	// AI stats endpoint
	api.HandleFunc("/ai/stats", r.handleStats).Methods(http.MethodGet)

	// Semantic search endpoint
	api.HandleFunc("/ai/search", r.handleSemanticSearch).Methods(http.MethodPost)

	// Initialize routes
	r.logger.Info("AI routes registered")
}

// handleAIQuery handles AI query requests
func (r *Router) handleAIQuery(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		r.handleError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if req.Text == "" {
		r.handleError(w, http.StatusBadRequest, "query text is required", nil)
		return
	}

	// Build query context
	queryContext := &QueryContext{
		UserID:      req.UserID,
		Query:       req.Text,
		ContextSize: req.ContextSize,
		Tools:       req.Tools,
		Filter:      req.Filter,
		SessionID:   req.SessionID,
		Timestamp:   time.Now(),
	}

	// Process query
	ctx := r.withRequestID(r.Context(), r)
	result, err := r.aiEngine.ProcessQuery(ctx, queryContext)
	if err != nil {
		r.handleError(w, http.StatusInternalServerError, "query processing failed", err)
		return
	}

	// Calculate processing time
	processingTime := time.Since(startTime).String()

	response := QueryResponse{
		ProcessedQuery: result.ProcessedQuery,
		Answer:         result.Answer,
		Sources:        result.Sources,
		Metadata:       result.Metadata,
		TokensUsed:     result.Metadata["tokens_used"].(int),
		ProcessingTime: processingTime,
	}

	r.sendJSON(w, http.StatusOK, response)
	r.logger.WithFields(logrus.Fields{
		"user_id":   req.UserID,
		"tools":     req.Tools,
		"time":      processingTime,
	}).Info("Query processed successfully")
}

// handleRecommendations handles recommendations requests
func (r *Router) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var req RecommendationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		r.handleError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if req.UserID == "" {
		r.handleError(w, http.StatusBadRequest, "user_id is required", nil)
		return
	}

	if req.Query == "" {
		r.handleError(w, http.StatusBadRequest, "query is required", nil)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	} else if req.Limit > 100 {
		req.Limit = 100
	}

	// Generate recommendations
	ctx := r.withRequestID(r.Context(), r)
	recommendations, err := r.aiEngine.GenerateRecommendations(ctx, req.UserID, req.Query, req.Limit)
	if err != nil {
		r.handleError(w, http.StatusInternalServerError, "recommendation generation failed", err)
		return
	}

	// Calculate query time
	queryTime := time.Since(startTime).String()

	response := RecommendationsResponse{
		Recommendations: recommendations,
		TotalFound:      len(recommendations),
		QueryTime:       queryTime,
		UserID:          req.UserID,
	}

	r.sendJSON(w, http.StatusOK, response)
	r.logger.WithFields(logrus.Fields{
		"user_id":     req.UserID,
		"limit":       req.Limit,
		"total_found": len(recommendations),
		"time":        queryTime,
	}).Info("Recommendations generated successfully")
}

// handleSemanticSearch handles semantic search requests
func (r *Router) handleSemanticSearch(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var searchReq struct {
		Query   string            `json:"query"`
		TopK    int               `json:"top_k"`
		Filter  map[string]interface{} `json:"filter"`
		UserID  string            `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		r.handleError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if searchReq.Query == "" {
		r.handleError(w, http.StatusBadRequest, "query is required", nil)
		return
	}

	if searchReq.TopK <= 0 {
		searchReq.TopK = 10
	} else if searchReq.TopK > 100 {
		searchReq.TopK = 100
	}

	// Perform semantic search
	ctx := r.withRequestID(r.Context(), r)
	results, err := r.aiEngine.SemanticSearch(ctx, searchReq.Query, searchReq.TopK, searchReq.Filter)
	if err != nil {
		r.handleError(w, http.StatusInternalServerError, "semantic search failed", err)
		return
	}

	// Calculate query time
	queryTime := time.Since(startTime).String()

	response := map[string]interface{}{
		"query":       searchReq.Query,
		"results":     results,
		"total":       len(results),
		"query_time":  queryTime,
		"user_id":     searchReq.UserID,
	}

	r.sendJSON(w, http.StatusOK, response)
}

// handleHealth handles health check requests
func (r *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := r.getHealthStatus()
	r.sendJSON(w, http.StatusOK, status)
}

// handleStats handles stats requests
func (r *Router) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := r.aiEngine.GetRecommendationStats()

	if r.openAIClient != nil {
		stats["openai"] = r.openAIClient.GetStats()
	}

	if r.pgVector != nil {
		count, err := r.pgVector.GetEmbeddingCount(r.Context())
		if err == nil {
			stats["total_embeddings"] = count
		}
	}

	r.sendJSON(w, http.StatusOK, stats)
}

// getHealthStatus returns the health status of AI services
func (r *Router) getHealthStatus() HealthStatus {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]string),
		Metrics:   r.aiEngine.GetRecommendationStats(),
	}

	if r.openAIClient != nil {
		status.Services["openai"] = "connected"
	} else {
		status.Services["openai"] = "mock"
	}

	if r.pgVector != nil {
		status.Services["pgvector"] = "connected"
	} else {
		status.Services["pgvector"] = "disabled"
	}

	return status
}

// withRequestID adds a request ID to the context
func (r *Router) withRequestID(ctx context.Context, httpR *http.Request) context.Context {
	requestID := httpR.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return context.WithValue(ctx, "request_id", requestID)
}

// sendJSON sends a JSON response
func (r *Router) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", w.Header().Get("X-Request-ID"))

	if err := json.NewEncoder(w).Encode(data); err != nil {
		r.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// handleError sends an error response
func (r *Router) handleError(w http.ResponseWriter, status int, message string, err error) {
	errorResponse := map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
		"status": status,
	}

	if err != nil {
		errorResponse["error"]["detail"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse)

	r.logger.WithFields(logrus.Fields{
		"status": status,
		"error":  message,
	}).Error("AI request failed")
}

// RegisterServiceEmbedding registers a service embedding for the AI engine
func (r *Router) RegisterServiceEmbedding(serviceID, serviceName, category string, tags []string, metadata map[string]string) error {
	ctx := r.Context()
	embedding, err := r.aiEngine.GenerateEmbedding(ctx, serviceID, serviceName, tags, category, metadata)
	if err != nil {
		return err
	}

	r.logger.WithFields(logrus.Fields{
		"service_id": serviceID,
		"service_name": serviceName,
		"embedding_id": embedding.ID,
	}).Info("Service embedding registered")

	return nil
}

// GetEmbedding retrieves an embedding by ID
func (r *Router) GetEmbedding(id string) (*VectorResult, error) {
	if r.pgVector == nil {
		return nil, fmt.Errorf("pgvector not available")
	}

	return r.pgVector.GetEmbedding(r.Context(), id)
}

// SearchSimilar finds similar embeddings
func (r *Router) SearchSimilar(queryVector []float32, topK int, filter map[string]interface{}) ([]VectorResult, error) {
	if r.pgVector == nil {
		return nil, fmt.Errorf("pgvector not available")
	}

	return r.pgVector.SearchSimilar(r.Context(), queryVector, topK, filter)
}
