package ai

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// OpenAIClient interface for OpenAI API communication
type OpenAIClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Complete(ctx context.Context, prompt string) (string, error)
}

// PGVectorClient interface for PostgreSQL pgvector operations
type PGVectorClient interface {
	StoreEmbedding(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error
	SearchSimilar(ctx context.Context, queryVector []float32, topK int, filter map[string]interface{}) ([]VectorResult, error)
	DeleteEmbedding(ctx context.Context, id string) error
	GetEmbedding(ctx context.Context, id string) (*VectorResult, error)
}

// VectorResult represents a vector search result
type VectorResult struct {
	ID         string                 `json:"id"`
	Score      float32                `json:"score"`
	Metadata   map[string]interface{} `json:"metadata"`
	Vector     []float32              `json:"-"`
}

// UserContext tracks user preferences and history
type UserContext struct {
	ID              string                 `json:"id"`
	Preferences     map[string]interface{} `json:"preferences"`
	RecentQueries   []string               `json:"recent_queries"`
	Interactions    map[string]int         `json:"interactions"` // service_id -> interaction_count
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ServiceEmbedding represents an embedding for a service
type ServiceEmbedding struct {
	ID           string            `json:"id"`
	Vector       []float32         `json:"vector"`
	ServiceID    string            `json:"service_id"`
	ServiceName  string            `json:"service_name"`
	Tags         []string          `json:"tags"`
	Category     string            `json:"category"`
	Metadata     map[string]string `json:"metadata"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// Recommendation represents an AI recommendation
type Recommendation struct {
	ServiceID      string            `json:"service_id"`
	ServiceName    string            `json:"service_name"`
	Reason         string            `json:"reason"`
	Confidence     float32           `json:"confidence"`
	SimilarityScore float32          `json:"similarity_score"`
	Category       string            `json:"category"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// QueryContext contains the context for an AI query
type QueryContext struct {
	UserID       string            `json:"user_id"`
	Query        string            `json:"query"`
	ContextSize  int               `json:"context_size"`
	Tools        []string          `json:"tools"`
	Filter       map[string]string `json:"filter"`
	SessionID    string            `json:"session_id"`
	Timestamp    time.Time         `json:"timestamp"`
}

// QueryResult represents the result of an AI query
type QueryResult struct {
	ProcessedQuery string                 `json:"processed_query"`
	Answer         string                 `json:"answer"`
	Sources        []string               `json:"sources"`
	Metadata       map[string]interface{} `json:"metadata"`
	TokensUsed     int                    `json:"tokens_used"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// RecommendationEngine handles service recommendations
type RecommendationEngine struct {
	logger        *logrus.Logger
	userStore     map[string]*UserContext
	userMu        sync.RWMutex
	embeddingMu   sync.RWMutex
	serviceEmbed map[string]*ServiceEmbedding
	embeddingIdx  int // Counter for vector index positions
	openAI        OpenAIClient
	pgVector      PGVectorClient
	promptEngine  PromptEngine
}

// PromptEngine interface for generating prompts
type PromptEngine interface {
	BuildQueryPrompt(query *QueryContext) string
	BuildRecommendationPrompt(query string, context map[string]string) string
	BuildIntentPrompt(query string) string
}

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine(logger *logrus.Logger, openAI OpenAIClient, pgVector PGVectorClient) *RecommendationEngine {
	return &RecommendationEngine{
		logger:       logger.WithField("component", "ai_engine"),
		userStore:    make(map[string]*UserContext),
		serviceEmbed: make(map[string]*ServiceEmbedding),
		openAI:       openAI,
		pgVector:     pgVector,
		promptEngine: NewDefaultPromptEngine(),
	}
}

// GetOrCreateUserContext retrieves or creates a user context
func (e *RecommendationEngine) GetOrCreateUserContext(userID string) *UserContext {
	e.userMu.Lock()
	defer e.userMu.Unlock()

	if ctx, exists := e.userStore[userID]; exists {
		ctx.UpdatedAt = time.Now()
		return ctx
	}

	ctx := &UserContext{
		ID:          userID,
		Preferences: make(map[string]interface{}),
		Interactions: make(map[string]int),
		RecentQueries: make([]string, 0),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	e.userStore[userID] = ctx
	return ctx
}

// TrackUserInteraction records a user interaction with a service
func (e *RecommendationEngine) TrackUserInteraction(userID, serviceID string) {
	e.userMu.Lock()
	defer e.userMu.Unlock()

	ctx, exists := e.userStore[userID]
	if !exists {
		ctx = e.GetOrCreateUserContext(userID)
	}

	ctx.Interactions[serviceID]++
	ctx.UpdatedAt = time.Now()
}

// RecordUserQuery records a user query for context tracking
func (e *RecommendationEngine) RecordUserQuery(userID, query string) {
	e.userMu.Lock()
	defer e.userMu.Unlock()

	ctx, exists := e.userStore[userID]
	if !exists {
		ctx = e.GetOrCreateUserContext(userID)
	}

	ctx.RecentQueries = append(ctx.RecentQueries, query)
	if len(ctx.RecentQueries) > 10 {
		ctx.RecentQueries = ctx.RecentQueries[len(ctx.RecentQueries)-10:]
	}
	ctx.UpdatedAt = time.Now()
}

// GenerateEmbedding generates an embedding for a service
func (e *RecommendationEngine) GenerateEmbedding(ctx context.Context, serviceID, serviceName string, tags []string, category string, metadata map[string]string) (*ServiceEmbedding, error) {
	e.embeddingMu.Lock()
	defer e.embeddingMu.Unlock()

	// Combine service info into a text representation
	text := fmt.Sprintf("%s %s", serviceName, category)
	text += " " + fmt.Sprintf("%v", tags)
	for _, v := range metadata {
		text += " " + v
	}

	// Generate embedding using OpenAI
	vector, err := e.openAI.Embed(ctx, text)
	if err != nil {
		e.logger.WithError(err).Error("Failed to generate embedding")
		return nil, err
	}

	embedding := &ServiceEmbedding{
		ID:          fmt.Sprintf("%d", e.embeddingIdx),
		Vector:      vector,
		ServiceID:   serviceID,
		ServiceName: serviceName,
		Tags:        tags,
		Category:    category,
		Metadata:    metadata,
		UpdatedAt:   time.Now(),
	}

	e.embeddingIdx++
	e.serviceEmbed[serviceID] = embedding

	// Store in vector database
	if e.pgVector != nil {
		metadataMap := make(map[string]interface{})
		for k, v := range metadata {
			metadataMap[k] = v
		}
		metadataMap["tags"] = tags
		metadataMap["category"] = category
		metadataMap["service_id"] = serviceID

		if err := e.pgVector.StoreEmbedding(ctx, embedding.ID, vector, metadataMap); err != nil {
			e.logger.WithError(err).Error("Failed to store embedding in vector DB")
		}
	}

	return embedding, nil
}

// SemanticSearch performs a semantic search for services
func (e *RecommendationEngine) SemanticSearch(ctx context.Context, query string, topK int, filters map[string]interface{}) ([]VectorResult, error) {
	e.logger.WithFields(logrus.Fields{
		"query":  query,
		"top_k":  topK,
		"filters": filters,
	}).Debug("Semantic search")

	// Generate embedding for the query
	queryVector, err := e.openAI.Embed(ctx, query)
	if err != nil {
		e.logger.WithError(err).Error("Failed to generate query embedding")
		return nil, err
	}

	// Search using vector database
	if e.pgVector != nil {
		results, err := e.pgVector.SearchSimilar(ctx, queryVector, topK, filters)
		if err != nil {
			e.logger.WithError(err).Error("Failed to search vector DB")
			return nil, err
		}
		return results, nil
	}

	// Fallback to local search if vector DB not available
	return e.localSemanticSearch(ctx, queryVector, topK, filters)
}

// localSemanticSearch performs semantic search using in-memory embeddings
func (e *RecommendationEngine) localSemanticSearch(ctx context.Context, queryVector []float32, topK int, filters map[string]interface{}) ([]VectorResult, error) {
	e.embeddingMu.RLock()
	defer e.embeddingMu.RUnlock()

	type scoredResult struct {
		result VectorResult
		score  float32
	}

	var scored []scoredResult

	for _, embedding := range e.serviceEmbed {
		// Apply filters
		if !matchesFilters(embedding, filters) {
			continue
		}

		// Calculate cosine similarity
		score := cosineSimilarity(embedding.Vector, queryVector)

		scored = append(scored, scoredResult{
			result: VectorResult{
				ID:       embedding.ID,
				Score:    score,
				Metadata: embedding.Metadata,
				Vector:   embedding.Vector,
			},
			score: score,
		})
	}

	// Sort by score descending
	scored = sortByScoreDesc(scored)

	// Return top K results
	var results []VectorResult
	for i := 0; i < len(scored) && i < topK; i++ {
		results = append(results, scored[i].result)
	}

	return results, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(float64(normA)) * math.Sqrt(float64(normB)))
}

// sortByScoreDesc sorts scored results by score in descending order
func sortByScoreDesc(scored []scoredResult) []scoredResult {
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	return scored
}

// matchesFilters checks if an embedding matches the given filters
func matchesFilters(embedding *ServiceEmbedding, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "category":
			if embedding.Category != value {
				return false
			}
		case "tags":
			tagsList, ok := value.([]string)
			if !ok {
				return false
			}
			hasTag := false
			for _, tag := range tagsList {
				for _, embeddingTag := range embedding.Tags {
					if tag == embeddingTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				return false
			}
		case "service_id":
			if embedding.ServiceID != value {
				return false
			}
		}
	}
	return true
}

// GenerateRecommendations generates context-aware recommendations
func (e *RecommendationEngine) GenerateRecommendations(ctx context.Context, userID string, query string, limit int) ([]Recommendation, error) {
	e.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"query":   query,
		"limit":   limit,
	}).Info("Generating recommendations")

	// Get user context
	userCtx := e.GetOrCreateUserContext(userID)

	// Perform semantic search
	results, err := e.SemanticSearch(ctx, query, limit+5, nil)
	if err != nil {
		e.logger.WithError(err).Error("Semantic search failed")
		return nil, err
	}

	// Generate recommendations with ranking
	var recommendations []Recommendation
	for _, result := range results {
		// Boost scores based on user interactions
		boost := 1.0
		if interactionCount := userCtx.Interactions[result.ID]; interactionCount > 0 {
			boost += float32(interactionCount) * 0.1
		}

		adjustedScore := result.Score * boost
		if adjustedScore > 0.5 {
			recommendation := Recommendation{
				ServiceID:      result.Metadata["service_id"].(string),
				ServiceName:    result.Metadata["service_name"].(string),
				Reason:         e.generateReason(result, userCtx),
				Confidence:     adjustedScore,
				SimilarityScore: result.Score,
				Category:       result.Metadata["category"].(string),
				Metadata:       result.Metadata,
			}
			recommendations = append(recommendations, recommendation)

			if len(recommendations) >= limit {
				break
			}
		}
	}

	return recommendations, nil
}

// generateReason generates a natural language reason for a recommendation
func (e *RecommendationEngine) generateReason(result VectorResult, userCtx *UserContext) string {
	// Use prompt engine to generate reason
	prompt := e.promptEngine.BuildRecommendationPrompt(
		fmt.Sprintf("service %s", result.Metadata["service_name"]),
		result.Metadata,
	)

	if e.openAI != nil {
		if reason, err := e.openAI.Complete(context.Background(), prompt); err == nil {
			return reason
		}
	}

	// Fallback: generate simple reason
	category := result.Metadata["category"].(string)
	tags := result.Metadata["tags"].([]string)
	return fmt.Sprintf("Recommended based on category '%s' and tags: %v", category, tags)
}

// ProcessQuery processes a natural language query
func (e *RecommendationEngine) ProcessQuery(ctx context.Context, query *QueryContext) (*QueryResult, error) {
	startTime := time.Now()
	e.logger.WithFields(logrus.Fields{
		"user_id": query.UserID,
		"query":   query.Query,
		"tools":   query.Tools,
	}).Info("Processing AI query")

	// Record query for context
	e.RecordUserQuery(query.UserID, query.Query)

	// Build prompt
	prompt := e.promptEngine.BuildQueryPrompt(query)

	// Get LLM response
	var answer string
	var err error
	if e.openAI != nil {
		answer, err = e.openAI.Complete(ctx, prompt)
		if err != nil {
			e.logger.WithError(err).Error("LLM completion failed")
			return nil, err
		}
	} else {
		// Fallback: use simple keyword matching
		answer = e.processQueryLocally(query)
	}

	// Extract sources (services that were referenced)
	sources := e.extractSources(query.Query, answer)

	result := &QueryResult{
		ProcessedQuery: prompt,
		Answer:         answer,
		Sources:        sources,
		Metadata: map[string]interface{}{
			"query_id":     fmt.Sprintf("%d", time.Now().UnixNano()),
			"model":        "openai",
			"tools_used":   query.Tools,
		},
		ProcessingTime: time.Since(startTime),
	}

	return result, nil
}

// processQueryLocally processes a query without LLM
func (e *RecommendationEngine) processQueryLocally(query *QueryContext) string {
	// Simple keyword matching fallback
	queryText := query.Query

	// Search for matching services
	var matchingServices []string
	for _, embedding := range e.serviceEmbed {
		if containsKeyword(queryText, embedding.ServiceName) ||
			containsKeyword(queryText, embedding.Category) ||
			containsAnyKeyword(queryText, embedding.Tags) {
			matchingServices = append(matchingServices, embedding.ServiceName)
		}
	}

	if len(matchingServices) > 0 {
		return fmt.Sprintf("Found matching services: %v", matchingServices)
	}

	return "No matching services found. Try rephrasing your query."
}

// containsKeyword checks if query contains any keyword from service info
func containsKeyword(query, text string) bool {
	queryLower := fmt.Sprintf(" %s ", query)
	textLower := fmt.Sprintf(" %s ", text)
	return len(query) > 0 && len(text) > 0 &&
		(len(queryLower) >= len(textLower) && containsSubstring(queryLower, textLower) ||
		 len(textLower) >= len(queryLower) && containsSubstring(textLower, queryLower))
}

// containsAnyKeyword checks if query contains any of the given tags
func containsAnyKeyword(query string, tags []string) bool {
	for _, tag := range tags {
		if containsKeyword(query, tag) {
			return true
		}
	}
	return false
}

// containsSubstring checks if big contains small as substring
func containsSubstring(big, small string) bool {
	for i := 0; i <= len(big)-len(small); i++ {
		if big[i:i+len(small)] == small {
			return true
		}
	}
	return false
}

// extractSources extracts service IDs from query and answer
func (e *RecommendationEngine) extractSources(query, answer string) []string {
	// Simple implementation - in production would use NLP
	var sources []string
	for serviceID := range e.serviceEmbed {
		if containsKeyword(query, serviceID) || containsKeyword(answer, serviceID) {
			sources = append(sources, serviceID)
		}
	}
	return sources
}

// GetQueryIntent extracts intent from a query
func (e *RecommendationEngine) GetQueryIntent(ctx context.Context, query string) (string, error) {
	prompt := e.promptEngine.BuildIntentPrompt(query)

	if e.openAI != nil {
		intent, err := e.openAI.Complete(ctx, prompt)
		if err != nil {
			return "", err
		}
		return intent, nil
	}

	// Fallback: simple keyword-based intent detection
	return e.detectIntentLocally(query), nil
}

// detectIntentLocally detects query intent without LLM
func (e *RecommendationEngine) detectIntentLocally(query string) string {
	queryLower := fmt.Sprintf(" %s ", query)

	if containsSubstring(queryLower, " find ") || containsSubstring(queryLower, " search ") {
		return "search"
	}
	if containsSubstring(queryLower, " recommend ") {
		return "recommend"
	}
	if containsSubstring(queryLower, " how ") || containsSubstring(queryLower, " what ") || containsSubstring(queryLower, " why ") {
		return "question"
	}
	if containsSubstring(queryLower, "create ") || containsSubstring(queryLower, " deploy ") {
		return "action"
	}
	return "search"
}

// OptimizeContext reduces context to fit within the token window
func (e *RecommendationEngine) OptimizeContext(ctx context.Context, data map[string]interface{}, maxTokens int) map[string]interface{} {
	// In production, would use token counting and truncation
	// For now, return as-is with metadata
	data["optimized"] = true
	data["max_tokens"] = maxTokens
	return data
}

// GenerateEmbeddingHash creates a hash from an embedding vector
func GenerateEmbeddingHash(vector []float32) string {
	hashBytes := make([]byte, 0)
	for _, v := range vector {
		hashBytes = append(hashBytes, byte(v*100)%256)
	}
	return hex.EncodeToString(hashBytes[:16])
}

// GetRecommendationStats returns statistics about the recommendation engine
func (e *RecommendationEngine) GetRecommendationStats() map[string]interface{} {
	e.userMu.RLock()
	defer e.userMu.RUnlock()

	return map[string]interface{}{
		"total_users":      len(e.userStore),
		"total_embeddings": len(e.serviceEmbed),
		"embedding_index":  e.embeddingIdx,
	}
}

// SeedRandom initializes random number generator
func init() {
	rand.Seed(time.Now().UnixNano())
}
