package ai

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// OpenAIClient defines the AI operations used by the engine.
type OpenAIClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Complete(ctx context.Context, prompt string) (string, error)
	GetStats() map[string]interface{}
}

// PGVectorClient defines the persistence/search operations for embeddings.
type PGVectorClient interface {
	StoreEmbedding(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error
	SearchSimilar(ctx context.Context, queryVector []float32, topK int, filter map[string]interface{}) ([]VectorResult, error)
	DeleteEmbedding(ctx context.Context, id string) error
	GetEmbedding(ctx context.Context, id string) (*VectorResult, error)
}

// VectorResult represents a vector search result.
type VectorResult struct {
	ID       string                 `json:"id"`
	Score    float32                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
	Vector   []float32              `json:"-"`
}

// UserContext tracks user preferences and query history.
type UserContext struct {
	ID            string                 `json:"id"`
	Preferences   map[string]interface{} `json:"preferences"`
	RecentQueries []string               `json:"recent_queries"`
	Interactions  map[string]int         `json:"interactions"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ServiceEmbedding represents a service embedding.
type ServiceEmbedding struct {
	ID          string            `json:"id"`
	Vector      []float32         `json:"vector"`
	ServiceID   string            `json:"service_id"`
	ServiceName string            `json:"service_name"`
	Tags        []string          `json:"tags"`
	Category    string            `json:"category"`
	Metadata    map[string]string `json:"metadata"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Recommendation represents a generated recommendation.
type Recommendation struct {
	ServiceID       string                 `json:"service_id"`
	ServiceName     string                 `json:"service_name"`
	Reason          string                 `json:"reason"`
	Confidence      float32                `json:"confidence"`
	SimilarityScore float32                `json:"similarity_score"`
	Category        string                 `json:"category"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// QueryContext contains the context for a natural-language request.
type QueryContext struct {
	UserID      string            `json:"user_id"`
	Query       string            `json:"query"`
	ContextSize int               `json:"context_size"`
	Tools       []string          `json:"tools"`
	Filter      map[string]string `json:"filter"`
	SessionID   string            `json:"session_id"`
	Timestamp   time.Time         `json:"timestamp"`
}

// QueryResult represents the processed query response.
type QueryResult struct {
	ProcessedQuery string                 `json:"processed_query"`
	Answer         string                 `json:"answer"`
	Sources        []string               `json:"sources"`
	Metadata       map[string]interface{} `json:"metadata"`
	TokensUsed     int                    `json:"tokens_used"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

type scoredResult struct {
	result VectorResult
	score  float32
}

// PromptEngine builds prompts for the AI client.
type PromptEngine interface {
	BuildQueryPrompt(query *QueryContext) string
	BuildRecommendationPrompt(query string, context map[string]string) string
	BuildIntentPrompt(query string) string
}

// RecommendationEngine handles query processing and recommendations.
type RecommendationEngine struct {
	logger       *logrus.Logger
	userStore    map[string]*UserContext
	userMu       sync.RWMutex
	embeddingMu  sync.RWMutex
	serviceEmbed map[string]*ServiceEmbedding
	embeddingIdx int
	openAI       OpenAIClient
	pgVector     PGVectorClient
	promptEngine PromptEngine
}

// NewRecommendationEngine creates a new recommendation engine.
func NewRecommendationEngine(logger *logrus.Logger, openAI OpenAIClient, pgVector PGVectorClient) *RecommendationEngine {
	if openAI == nil {
		openAI = NewMockOpenAIClient()
	}
	if isNilInterface(pgVector) {
		pgVector = nil
	}

	return &RecommendationEngine{
		logger:       logger,
		userStore:    make(map[string]*UserContext),
		serviceEmbed: make(map[string]*ServiceEmbedding),
		openAI:       openAI,
		pgVector:     pgVector,
		promptEngine: NewDefaultPromptEngine(),
	}
}

func (e *RecommendationEngine) getOrCreateUserContextLocked(userID string) *UserContext {
	if ctx, exists := e.userStore[userID]; exists {
		return ctx
	}

	ctx := &UserContext{
		ID:            userID,
		Preferences:   make(map[string]interface{}),
		Interactions:  make(map[string]int),
		RecentQueries: make([]string, 0, 10),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	e.userStore[userID] = ctx
	return ctx
}

// GetOrCreateUserContext retrieves or creates a context for the given user.
func (e *RecommendationEngine) GetOrCreateUserContext(userID string) *UserContext {
	e.userMu.Lock()
	defer e.userMu.Unlock()
	return e.getOrCreateUserContextLocked(userID)
}

// TrackUserInteraction records user interaction with a service.
func (e *RecommendationEngine) TrackUserInteraction(userID, serviceID string) {
	e.userMu.Lock()
	defer e.userMu.Unlock()

	ctx := e.getOrCreateUserContextLocked(userID)
	ctx.Interactions[serviceID]++
	ctx.UpdatedAt = time.Now()
}

// RecordUserQuery records a user query for later context.
func (e *RecommendationEngine) RecordUserQuery(userID, query string) {
	e.userMu.Lock()
	defer e.userMu.Unlock()

	ctx := e.getOrCreateUserContextLocked(userID)
	ctx.RecentQueries = append(ctx.RecentQueries, query)
	if len(ctx.RecentQueries) > 10 {
		ctx.RecentQueries = ctx.RecentQueries[len(ctx.RecentQueries)-10:]
	}
	ctx.UpdatedAt = time.Now()
}

// GenerateEmbedding generates and stores a service embedding.
func (e *RecommendationEngine) GenerateEmbedding(ctx context.Context, serviceID, serviceName string, tags []string, category string, metadata map[string]string) (*ServiceEmbedding, error) {
	text := strings.TrimSpace(strings.Join([]string{serviceName, category, strings.Join(tags, " ")}, " "))
	for _, v := range metadata {
		text += " " + v
	}

	vector, err := e.openAI.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	embedding := &ServiceEmbedding{
		ID:          serviceID,
		Vector:      vector,
		ServiceID:   serviceID,
		ServiceName: serviceName,
		Tags:        append([]string(nil), tags...),
		Category:    category,
		Metadata:    cloneStringMap(metadata),
		UpdatedAt:   time.Now(),
	}

	e.embeddingMu.Lock()
	e.serviceEmbed[serviceID] = embedding
	e.embeddingIdx++
	e.embeddingMu.Unlock()

	if e.pgVector != nil {
		meta := make(map[string]interface{}, len(metadata)+3)
		for k, v := range metadata {
			meta[k] = v
		}
		meta["tags"] = append([]string(nil), tags...)
		meta["category"] = category
		meta["service_id"] = serviceID
		_ = e.pgVector.StoreEmbedding(ctx, embedding.ID, vector, meta)
	}

	return embedding, nil
}

// SemanticSearch performs a search over stored embeddings.
func (e *RecommendationEngine) SemanticSearch(ctx context.Context, query string, topK int, filters map[string]interface{}) ([]VectorResult, error) {
	if topK <= 0 {
		topK = 10
	}

	queryVector, err := e.openAI.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	if e.pgVector != nil {
		if results, err := e.pgVector.SearchSimilar(ctx, queryVector, topK, filters); err == nil {
			return results, nil
		}
	}

	return e.localSemanticSearch(queryVector, topK, filters), nil
}

func (e *RecommendationEngine) localSemanticSearch(queryVector []float32, topK int, filters map[string]interface{}) []VectorResult {
	e.embeddingMu.RLock()
	defer e.embeddingMu.RUnlock()

	scored := make([]scoredResult, 0, len(e.serviceEmbed))
	for _, embedding := range e.serviceEmbed {
		if !matchesFilters(embedding, filters) {
			continue
		}

		scored = append(scored, scoredResult{
			result: VectorResult{
				ID:       embedding.ID,
				Score:    cosineSimilarity(embedding.Vector, queryVector),
				Metadata: embeddingToMetadata(embedding),
				Vector:   append([]float32(nil), embedding.Vector...),
			},
			score: cosineSimilarity(embedding.Vector, queryVector),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	results := make([]VectorResult, 0, min(topK, len(scored)))
	for i := 0; i < len(scored) && i < topK; i++ {
		results = append(results, scored[i].result)
	}
	return results
}

// GenerateRecommendations generates ranked recommendations.
func (e *RecommendationEngine) GenerateRecommendations(ctx context.Context, userID string, query string, limit int) ([]Recommendation, error) {
	if limit <= 0 {
		limit = 10
	}

	userCtx := e.GetOrCreateUserContext(userID)
	results, err := e.SemanticSearch(ctx, query, limit+5, nil)
	if err != nil {
		return nil, err
	}

	recommendations := make([]Recommendation, 0, limit)
	for _, result := range results {
		serviceID := metadataString(result.Metadata, "service_id", result.ID)
		serviceName := metadataString(result.Metadata, "service_name", serviceID)
		category := metadataString(result.Metadata, "category", "general")

		boost := float32(1.0)
		if n := userCtx.Interactions[serviceID]; n > 0 {
			boost += float32(n) * 0.1
		}

		confidence := result.Score * boost
		recommendations = append(recommendations, Recommendation{
			ServiceID:       serviceID,
			ServiceName:     serviceName,
			Reason:          e.generateReason(result, query),
			Confidence:      confidence,
			SimilarityScore: result.Score,
			Category:        category,
			Metadata:        result.Metadata,
		})

		if len(recommendations) >= limit {
			break
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, e.heuristicRecommendation(query))
	}

	return recommendations, nil
}

func (e *RecommendationEngine) heuristicRecommendation(query string) Recommendation {
	lower := strings.ToLower(query)
	serviceID := "demo-platform"
	serviceName := "Demo Platform"
	category := "platform"
	reason := "This is the default platform recommendation for an empty catalog."

	switch {
	case strings.Contains(lower, "database"):
		serviceID = "demo-database"
		serviceName = "Demo Database Service"
		category = "data"
		reason = "A database-oriented request benefits from a managed data service with clear ownership and predictable operations."
	case strings.Contains(lower, "auth"), strings.Contains(lower, "identity"), strings.Contains(lower, "login"):
		serviceID = "demo-auth"
		serviceName = "Demo Identity Service"
		category = "security"
		reason = "Identity-related work maps well to a centralized auth service with strong policy enforcement."
	}

	return Recommendation{
		ServiceID:       serviceID,
		ServiceName:     serviceName,
		Reason:          reason,
		Confidence:      0.55,
		SimilarityScore: 0.55,
		Category:        category,
		Metadata: map[string]interface{}{
			"query":  query,
			"source": "heuristic",
		},
	}
}

func (e *RecommendationEngine) generateReason(result VectorResult, query string) string {
	serviceName := metadataString(result.Metadata, "service_name", result.ID)
	category := metadataString(result.Metadata, "category", "general")
	tags := metadataStrings(result.Metadata, "tags")

	if e.openAI != nil {
		prompt := e.promptEngine.BuildRecommendationPrompt(
			fmt.Sprintf("service %s", serviceName),
			map[string]string{
				"query":    query,
				"category": category,
				"tags":     strings.Join(tags, ", "),
			},
		)
		if reason, err := e.openAI.Complete(context.Background(), prompt); err == nil && strings.TrimSpace(reason) != "" {
			return reason
		}
	}

	if len(tags) > 0 {
		return fmt.Sprintf("Matches category %q and tags %v", category, tags)
	}
	return fmt.Sprintf("Relevant to %q based on category %q", serviceName, category)
}

// ProcessQuery processes a natural-language query into a response.
func (e *RecommendationEngine) ProcessQuery(ctx context.Context, query *QueryContext) (*QueryResult, error) {
	start := time.Now()
	e.RecordUserQuery(query.UserID, query.Query)

	prompt := e.promptEngine.BuildQueryPrompt(query)
	answer, err := e.openAI.Complete(ctx, prompt)
	if err != nil || strings.TrimSpace(answer) == "" {
		answer = e.processQueryLocally(query)
	}

	result := &QueryResult{
		ProcessedQuery: prompt,
		Answer:         answer,
		Sources:        e.extractSources(query.Query, answer),
		Metadata: map[string]interface{}{
			"query_id":   fmt.Sprintf("%d", time.Now().UnixNano()),
			"model":      e.openAI.GetStats()["model"],
			"tools_used": append([]string(nil), query.Tools...),
		},
		TokensUsed:     tokenEstimate(prompt, answer),
		ProcessingTime: time.Since(start),
	}
	return result, nil
}

func (e *RecommendationEngine) processQueryLocally(query *QueryContext) string {
	queryText := strings.ToLower(query.Query)

	e.embeddingMu.RLock()
	defer e.embeddingMu.RUnlock()

	matches := make([]string, 0)
	for _, embedding := range e.serviceEmbed {
		if strings.Contains(queryText, strings.ToLower(embedding.ServiceName)) ||
			strings.Contains(queryText, strings.ToLower(embedding.Category)) ||
			containsAny(queryText, embedding.Tags) {
			matches = append(matches, embedding.ServiceName)
		}
	}

	if len(matches) == 0 {
		return "No matching services found. Try a more specific query."
	}
	return fmt.Sprintf("Found matching services: %s", strings.Join(matches, ", "))
}

func (e *RecommendationEngine) extractSources(query, answer string) []string {
	e.embeddingMu.RLock()
	defer e.embeddingMu.RUnlock()

	sources := make([]string, 0)
	seen := make(map[string]struct{})
	lower := strings.ToLower(query + " " + answer)
	for serviceID, embedding := range e.serviceEmbed {
		if strings.Contains(lower, strings.ToLower(serviceID)) || strings.Contains(lower, strings.ToLower(embedding.ServiceName)) {
			if _, ok := seen[serviceID]; !ok {
				seen[serviceID] = struct{}{}
				sources = append(sources, serviceID)
			}
		}
	}
	return sources
}

// GetQueryIntent derives the likely intent for a query.
func (e *RecommendationEngine) GetQueryIntent(ctx context.Context, query string) (string, error) {
	prompt := e.promptEngine.BuildIntentPrompt(query)
	if e.openAI != nil {
		if intent, err := e.openAI.Complete(ctx, prompt); err == nil && strings.TrimSpace(intent) != "" {
			return strings.TrimSpace(strings.ToLower(intent)), nil
		}
	}
	return e.detectIntentLocally(query), nil
}

func (e *RecommendationEngine) detectIntentLocally(query string) string {
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "recommend"), strings.Contains(lower, "suggest"):
		return "recommend"
	case strings.Contains(lower, "find"), strings.Contains(lower, "search"):
		return "search"
	case strings.Contains(lower, "how"), strings.Contains(lower, "what"), strings.Contains(lower, "why"):
		return "question"
	case strings.Contains(lower, "create"), strings.Contains(lower, "deploy"), strings.Contains(lower, "run"):
		return "action"
	default:
		return "search"
	}
}

// OptimizeContext returns a shallow copy annotated with optimization metadata.
func (e *RecommendationEngine) OptimizeContext(ctx context.Context, data map[string]interface{}, maxTokens int) map[string]interface{} {
	out := make(map[string]interface{}, len(data)+2)
	for k, v := range data {
		out[k] = v
	}
	out["optimized"] = true
	out["max_tokens"] = maxTokens
	return out
}

// GetRecommendationStats returns engine statistics.
func (e *RecommendationEngine) GetRecommendationStats() map[string]interface{} {
	e.userMu.RLock()
	defer e.userMu.RUnlock()

	return map[string]interface{}{
		"total_users":      len(e.userStore),
		"total_embeddings": len(e.serviceEmbed),
		"embedding_index":  e.embeddingIdx,
	}
}

// GenerateEmbeddingHash creates a stable hash of a vector.
func GenerateEmbeddingHash(vector []float32) string {
	hashBytes := make([]byte, 0, len(vector))
	for _, v := range vector {
		hashBytes = append(hashBytes, byte(v*100))
	}
	if len(hashBytes) > 16 {
		hashBytes = hashBytes[:16]
	}
	return hex.EncodeToString(hashBytes)
}

func matchesFilters(embedding *ServiceEmbedding, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "category":
			if fmt.Sprint(value) != embedding.Category {
				return false
			}
		case "service_id":
			if fmt.Sprint(value) != embedding.ServiceID {
				return false
			}
		case "tags":
			if !containsAny(fmt.Sprint(value), embedding.Tags) {
				if tagList, ok := value.([]string); ok && !sliceHasAny(tagList, embedding.Tags) {
					return false
				} else if !ok {
					return false
				}
			}
		}
	}
	return true
}

func containsAny(query string, values []string) bool {
	lower := strings.ToLower(query)
	for _, value := range values {
		if strings.Contains(lower, strings.ToLower(value)) {
			return true
		}
	}
	return false
}

func sliceHasAny(a, b []string) bool {
	for _, left := range a {
		for _, right := range b {
			if strings.EqualFold(left, right) {
				return true
			}
		}
	}
	return false
}

func embeddingToMetadata(embedding *ServiceEmbedding) map[string]interface{} {
	meta := make(map[string]interface{}, len(embedding.Metadata)+4)
	for k, v := range embedding.Metadata {
		meta[k] = v
	}
	meta["id"] = embedding.ID
	meta["service_id"] = embedding.ServiceID
	meta["service_name"] = embedding.ServiceName
	meta["category"] = embedding.Category
	meta["tags"] = append([]string(nil), embedding.Tags...)
	return meta
}

func metadataString(meta map[string]interface{}, key, fallback string) string {
	if meta == nil {
		return fallback
	}
	if v, ok := meta[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return fallback
}

func metadataStrings(meta map[string]interface{}, key string) []string {
	if meta == nil {
		return nil
	}
	raw, ok := meta[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		if s := fmt.Sprint(v); s != "" {
			return []string{s}
		}
		return nil
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func tokenEstimate(values ...string) int {
	total := 0
	for _, value := range values {
		total += len(strings.Fields(value))
	}
	if total == 0 {
		return 0
	}
	return total
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isNilInterface(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
