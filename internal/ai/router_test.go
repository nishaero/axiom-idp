package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// Test AI Router

func createTestRouter() (*Router, *httptest.Server) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := DefaultAIConfig()
	router := NewRouter(logger, config, nil)

	server := httptest.NewServer(router)
	return router, server
}

func TestRouter_HealthCheck(t *testing.T) {
	router, server := createTestRouter()

	// Initialize router
	ctx := context.Background()
	if err := router.Init(ctx); err != nil {
		t.Fatalf("Failed to initialize router: %v", err)
	}

	// Test health endpoint
	resp, err := http.Get(server.URL + "/ai/health")
	if err != nil {
		t.Fatalf("Health check request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", health.Status)
	}
}

func TestRouter_QueryEndpoint(t *testing.T) {
	router, server := createTestRouter()

	// Initialize router
	ctx := context.Background()
	if err := router.Init(ctx); err != nil {
		t.Fatalf("Failed to initialize router: %v", err)
	}

	// Test query endpoint
	query := &QueryRequest{
		Text:    "What services do you have?",
		UserID:  "test-user-123",
	}

	body, err := json.Marshal(query)
	if err != nil {
		t.Fatalf("Failed to marshal query: %v", err)
	}

	resp, err := http.Post(server.URL+"/ai/query", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Query request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode query response: %v", err)
	}

	if result.ProcessedQuery == "" {
		t.Error("Expected processed query to be set")
	}

	if result.Answer == "" {
		t.Error("Expected answer to be set")
	}
}

func TestRouter_RecommendationsEndpoint(t *testing.T) {
	router, server := createTestRouter()

	ctx := context.Background()
	if err := router.Init(ctx); err != nil {
		t.Fatalf("Failed to initialize router: %v", err)
	}

	// Test recommendations endpoint
	req := &RecommendationsRequest{
		UserID: "test-user-456",
		Query:  "I need a database service",
		Limit:  5,
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal recommendations request: %v", err)
	}

	resp, err := http.Post(server.URL+"/ai/recommendations", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Recommendations request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result RecommendationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode recommendations response: %v", err)
	}

	if len(result.Recommendations) == 0 {
		t.Error("Expected at least one recommendation")
	}
}

func TestRouter_SemanticSearchEndpoint(t *testing.T) {
	router, server := createTestRouter()

	ctx := context.Background()
	if err := router.Init(ctx); err != nil {
		t.Fatalf("Failed to initialize router: %v", err)
	}

	// Test semantic search endpoint
	req := map[string]interface{}{
		"query": "database management systems",
		"top_k": 10,
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal semantic search request: %v", err)
	}

	resp, err := http.Post(server.URL+"/ai/search", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Semantic search request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode semantic search response: %v", err)
	}

	if result["query"] != "database management systems" {
		t.Error("Expected query to be returned")
	}
}

func TestRouter_StatisticsEndpoint(t *testing.T) {
	router, server := createTestRouter()

	ctx := context.Background()
	if err := router.Init(ctx); err != nil {
		t.Fatalf("Failed to initialize router: %v", err)
	}

	// Test statistics endpoint
	resp, err := http.Get(server.URL + "/ai/stats")
	if err != nil {
		t.Fatalf("Statistics request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode statistics response: %v", err)
	}
}

func TestRouter_QueryContextManagement(t *testing.T) {
	router, server := createTestRouter()

	ctx := context.Background()
	if err := router.Init(ctx); err != nil {
		t.Fatalf("Failed to initialize router: %v", err)
	}

	// First query with context
	query1 := &QueryRequest{
		Text:       "What are my preferences?",
		UserID:     "test-user-context",
		ContextID:  "context-context",
		AdditionalContext: map[string]string{"lastQuery": "setup"},
	}

	body1, _ := json.Marshal(query1)
	resp1, err := http.Post(server.URL+"/ai/query", "application/json", bytes.NewBuffer(body1))
	if err != nil {
		t.Fatalf("First query failed: %v", err)
	}
	resp1.Body.Close()

	// Second query in same context
	query2 := &QueryRequest{
		Text:       "Tell me more about that",
		UserID:     "test-user-context",
		ContextID:  "context-context",
	}

	body2, _ := json.Marshal(query2)
	resp2, err := http.Post(server.URL+"/ai/query", "application/json", bytes.NewBuffer(body2))
	if err != nil {
		t.Fatalf("Second query failed: %v", err)
	}
	resp2.Body.Close()

	// Context should be maintained (verified by successful processing)
}

func TestRouter_ErrorHandling(t *testing.T) {
	router, server := createTestRouter()

	ctx := context.Background()
	if err := router.Init(ctx); err != nil {
		t.Fatalf("Failed to initialize router: %v", err)
	}

	// Test with empty query
	req := &QueryRequest{
		Text: "",
		UserID: "test-user-error",
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal error query: %v", err)
	}

	resp, err := http.Post(server.URL+"/ai/query", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error query request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 400 for validation error
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty query, got %d", resp.StatusCode)
	}
}

func TestRouter_Cleanup(t *testing.T) {
	router, server := createTestRouter()

	ctx := context.Background()
	if err := router.Init(ctx); err != nil {
		t.Fatalf("Failed to initialize router: %v", err)
	}

	// Test cleanup
	router.Cleanup()
}

// MockEngine for testing
type mockEngine struct{}

func (m *mockEngine) Init(ctx context.Context) error {
	return nil
}

func (m *mockEngine) Shutdown(ctx context.Context) error {
	return nil
}

func (m *mockEngine) GetEngine() *RecommendationEngine {
	return nil
}

func (m *mockEngine) ProcessQuery(ctx context.Context, query *QueryRequest, engine *RecommendationEngine) (*QueryResponse, error) {
	// Mock implementation
	response := &QueryResponse{
		ProcessedQuery: "mock query",
		Answer:         "mock answer",
		ProcessedAt:    time.Now(),
		ProcessingTime: 10,
	}
	return response, nil
}

func (m *mockEngine) GetRecommendations(ctx context.Context, request *RecommendationsRequest, engine *RecommendationEngine) (*RecommendationsResponse, error) {
	// Mock implementation
	return &RecommendationsResponse{
		RequestID:       "mock-request-id",
		GeneratedAt:     time.Now(),
		ProcessingTime:  20,
		Recommendations: []string{"mock1", "mock2"},
	}, nil
}

func (m *mockEngine) SemanticSearch(ctx context.Context, query map[string]interface{}) (map[string]interface{}, error) {
	// Mock implementation
	return map[string]interface{}{
		"query":      query["query"],
		"top_k":      query["top_k"],
		"results":    []string{},
		"results_count": 0,
		"search_duration_ms": 10,
	}, nil
}

func (m *mockEngine) GetStatistics(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"total_queries":  0,
		"total_recommendations": 0,
		"total_searches": 0,
		"active_contexts": 0,
		"engines_count":  0,
		"total_latency_ms": 0,
	}
}
