package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Router handles AI-related HTTP requests.
type Router struct {
	logger       *logrus.Logger
	config       *AIConfig
	aiEngine     *RecommendationEngine
	pgVector     *PGVectorEmbeddings
	openAIClient OpenAIClient
	promptEngine PromptEngine
	httpClient   *http.Client
	db           *sql.DB
	router       *mux.Router
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
}

// QueryRequest represents an AI query request.
type QueryRequest struct {
	Text              string            `json:"text"`
	ContextSize       int               `json:"context_size"`
	Tools             []string          `json:"tools"`
	UserID            string            `json:"user_id,omitempty"`
	Filter            map[string]string `json:"filter,omitempty"`
	SessionID         string            `json:"session_id,omitempty"`
	ContextID         string            `json:"context_id,omitempty"`
	AdditionalContext map[string]string `json:"additional_context,omitempty"`
}

// QueryResponse represents an AI query response.
type QueryResponse struct {
	ProcessedQuery string                 `json:"processed_query"`
	Answer         string                 `json:"answer"`
	Sources        []string               `json:"sources"`
	Metadata       map[string]interface{} `json:"metadata"`
	TokensUsed     int                    `json:"tokens_used"`
	ProcessingTime int                    `json:"processing_time"`
	ProcessedAt    time.Time              `json:"processed_at,omitempty"`
}

// RecommendationsRequest represents a recommendation request.
type RecommendationsRequest struct {
	UserID  string                 `json:"user_id"`
	Query   string                 `json:"query"`
	Limit   int                    `json:"limit,omitempty"`
	Filters map[string]interface{} `json:"filters,omitempty"`
}

// RecommendationsResponse represents recommendation results.
type RecommendationsResponse struct {
	Recommendations []Recommendation `json:"recommendations"`
	TotalFound      int              `json:"total_found"`
	QueryTime       string           `json:"query_time"`
	UserID          string           `json:"user_id"`
	RequestID       string           `json:"request_id,omitempty"`
	GeneratedAt     time.Time        `json:"generated_at,omitempty"`
	ProcessingTime  int              `json:"processing_time,omitempty"`
}

// HealthStatus represents AI health information.
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]string      `json:"services"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// NewRouter creates a new AI router.
func NewRouter(logger *logrus.Logger, config *AIConfig, db *sql.DB) *Router {
	ctx, cancel := context.WithCancel(context.Background())
	router := &Router{
		logger:     logger,
		config:     config,
		db:         db,
		router:     mux.NewRouter(),
		ctx:        ctx,
		cancel:     cancel,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	return router
}

// Context returns the router background context.
func (r *Router) Context() context.Context {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ctx
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.router.ServeHTTP(w, req)
}

// Init initializes the AI router and components.
func (r *Router) Init(ctx context.Context) error {
	r.logger.Info("initializing AI router")

	if r.config == nil {
		r.config = DefaultAIConfig()
	}

	if err := r.config.Validate(); err != nil {
		return err
	}

	if r.config.OpenAI.Enabled {
		r.openAIClient = NewOpenAIClient(r.logger, r.config.GetOpenAIClientConfig())
	} else {
		r.openAIClient = NewMockOpenAIClient()
	}

	if r.config.PGVector.Enabled && r.db != nil {
		r.pgVector = NewPGVectorEmbeddings(r.logger, r.db, r.config.GetPGVectorConfig())
		if err := r.pgVector.Initialize(ctx); err != nil {
			r.logger.WithError(err).Warn("pgvector initialization failed, continuing without vector DB")
			r.pgVector = nil
		}
	}

	r.promptEngine = NewDefaultPromptEngine()
	r.aiEngine = NewRecommendationEngine(r.logger, r.openAIClient, r.pgVector)
	r.setupRoutes()

	r.mu.Lock()
	r.ctx = ctx
	r.mu.Unlock()

	return nil
}

// SetupRoutes configures the AI routes.
func (r *Router) SetupRoutes(api *mux.Router) {
	r.router = api
	r.setupRoutes()
}

func (r *Router) setupRoutes() {
	r.router.HandleFunc("/ai/query", r.handleAIQuery).Methods(http.MethodPost)
	r.router.HandleFunc("/ai/recommendations", r.handleRecommendations).Methods(http.MethodPost)
	r.router.HandleFunc("/ai/health", r.handleHealth).Methods(http.MethodGet)
	r.router.HandleFunc("/ai/stats", r.handleStats).Methods(http.MethodGet)
	r.router.HandleFunc("/ai/search", r.handleSemanticSearch).Methods(http.MethodPost)
}

// Cleanup releases router resources.
func (r *Router) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Router) handleAIQuery(w http.ResponseWriter, req *http.Request) {
	start := time.Now()

	var payload QueryRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		r.handleError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if stringsTrim(payload.Text) == "" {
		r.handleError(w, http.StatusBadRequest, "query text is required", nil)
		return
	}

	queryContext := &QueryContext{
		UserID:      payload.UserID,
		Query:       payload.Text,
		ContextSize: payload.ContextSize,
		Tools:       payload.Tools,
		Filter:      payload.Filter,
		SessionID:   payload.SessionID,
		Timestamp:   time.Now(),
	}

	result, err := r.aiEngine.ProcessQuery(r.withRequestID(req.Context(), req), queryContext)
	if err != nil {
		r.handleError(w, http.StatusInternalServerError, "query processing failed", err)
		return
	}

	response := QueryResponse{
		ProcessedQuery: result.ProcessedQuery,
		Answer:         result.Answer,
		Sources:        result.Sources,
		Metadata:       result.Metadata,
		TokensUsed:     result.TokensUsed,
		ProcessingTime: int(time.Since(start).Milliseconds()),
		ProcessedAt:    time.Now(),
	}

	r.sendJSON(w, http.StatusOK, response)
}

func (r *Router) handleRecommendations(w http.ResponseWriter, req *http.Request) {
	start := time.Now()

	var payload RecommendationsRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		r.handleError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if stringsTrim(payload.UserID) == "" || stringsTrim(payload.Query) == "" {
		r.handleError(w, http.StatusBadRequest, "user_id and query are required", nil)
		return
	}
	if payload.Limit <= 0 {
		payload.Limit = 10
	}
	if payload.Limit > 100 {
		payload.Limit = 100
	}

	recommendations, err := r.aiEngine.GenerateRecommendations(r.withRequestID(req.Context(), req), payload.UserID, payload.Query, payload.Limit)
	if err != nil {
		r.handleError(w, http.StatusInternalServerError, "recommendation generation failed", err)
		return
	}

	response := RecommendationsResponse{
		Recommendations: recommendations,
		TotalFound:      len(recommendations),
		QueryTime:       time.Since(start).String(),
		UserID:          payload.UserID,
		RequestID:       req.Header.Get("X-Request-ID"),
		GeneratedAt:     time.Now(),
		ProcessingTime:  int(time.Since(start).Milliseconds()),
	}
	r.sendJSON(w, http.StatusOK, response)
}

func (r *Router) handleSemanticSearch(w http.ResponseWriter, req *http.Request) {
	start := time.Now()

	var payload struct {
		Query  string                 `json:"query"`
		TopK   int                    `json:"top_k"`
		Filter map[string]interface{} `json:"filter"`
		UserID string                 `json:"user_id"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		r.handleError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if stringsTrim(payload.Query) == "" {
		r.handleError(w, http.StatusBadRequest, "query is required", nil)
		return
	}
	if payload.TopK <= 0 {
		payload.TopK = 10
	}
	if payload.TopK > 100 {
		payload.TopK = 100
	}

	results, err := r.aiEngine.SemanticSearch(r.withRequestID(req.Context(), req), payload.Query, payload.TopK, payload.Filter)
	if err != nil {
		r.handleError(w, http.StatusInternalServerError, "semantic search failed", err)
		return
	}

	r.sendJSON(w, http.StatusOK, map[string]interface{}{
		"query":      payload.Query,
		"results":    results,
		"total":      len(results),
		"query_time": time.Since(start).String(),
		"user_id":    payload.UserID,
	})
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	r.sendJSON(w, http.StatusOK, r.getHealthStatus())
}

func (r *Router) handleStats(w http.ResponseWriter, req *http.Request) {
	stats := r.aiEngine.GetRecommendationStats()
	if r.openAIClient != nil {
		stats["openai"] = r.openAIClient.GetStats()
	}
	if r.pgVector != nil {
		if count, err := r.pgVector.GetEmbeddingCount(req.Context()); err == nil {
			stats["total_embeddings"] = count
		}
	}
	r.sendJSON(w, http.StatusOK, stats)
}

func (r *Router) getHealthStatus() HealthStatus {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  map[string]string{},
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

func (r *Router) withRequestID(ctx context.Context, req *http.Request) context.Context {
	requestID := req.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func (r *Router) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		r.logger.WithError(err).Error("failed to encode JSON response")
	}
}

func (r *Router) handleError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error":  map[string]string{"message": message},
		"status": status,
	}
	if err != nil {
		response["error"].(map[string]string)["detail"] = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

// RegisterServiceEmbedding registers an embedding for a catalog service.
func (r *Router) RegisterServiceEmbedding(serviceID, serviceName, category string, tags []string, metadata map[string]string) error {
	_, err := r.aiEngine.GenerateEmbedding(r.Context(), serviceID, serviceName, tags, category, metadata)
	return err
}

// GetEmbedding retrieves an embedding by ID.
func (r *Router) GetEmbedding(id string) (*VectorResult, error) {
	if r.pgVector == nil {
		return nil, fmt.Errorf("pgvector not available")
	}
	return r.pgVector.GetEmbedding(r.Context(), id)
}

// SearchSimilar finds similar embeddings.
func (r *Router) SearchSimilar(queryVector []float32, topK int, filter map[string]interface{}) ([]VectorResult, error) {
	if r.pgVector == nil {
		return nil, fmt.Errorf("pgvector not available")
	}
	return r.pgVector.SearchSimilar(r.Context(), queryVector, topK, filter)
}

type requestIDKey struct{}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}
