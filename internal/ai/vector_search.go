package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PGVectorEmbeddings implements vector search using an in-memory store.
type PGVectorEmbeddings struct {
	logger *logrus.Logger
	config PGVectorConfig

	mu      sync.RWMutex
	entries map[string]*VectorResult
}

// NewPGVectorEmbeddings creates a new pgvector embeddings client.
func NewPGVectorEmbeddings(logger *logrus.Logger, _ interface{}, config PGVectorConfig) *PGVectorEmbeddings {
	return &PGVectorEmbeddings{
		logger:  logger,
		config:  config,
		entries: make(map[string]*VectorResult),
	}
}

// Initialize prepares the in-memory store.
func (p *PGVectorEmbeddings) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.entries == nil {
		p.entries = make(map[string]*VectorResult)
	}
	return nil
}

// StoreEmbedding stores a service embedding.
func (p *PGVectorEmbeddings) StoreEmbedding(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	meta := cloneMetadata(metadata)
	meta["id"] = id
	p.entries[id] = &VectorResult{
		ID:       id,
		Score:    1,
		Metadata: meta,
		Vector:   append([]float32(nil), vector...),
	}
	return nil
}

// SearchSimilar performs a similarity search.
func (p *PGVectorEmbeddings) SearchSimilar(ctx context.Context, queryVector []float32, topK int, filters map[string]interface{}) ([]VectorResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if topK <= 0 {
		topK = 10
	}

	results := make([]VectorResult, 0, len(p.entries))
	for _, entry := range p.entries {
		if !matchesEmbeddingFilter(entry, filters) {
			continue
		}
		score := cosineSimilarity(entry.Vector, queryVector)
		results = append(results, VectorResult{
			ID:       entry.ID,
			Score:    score,
			Metadata: cloneMetadata(entry.Metadata),
			Vector:   append([]float32(nil), entry.Vector...),
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// DeleteEmbedding removes an embedding.
func (p *PGVectorEmbeddings) DeleteEmbedding(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.entries, id)
	return nil
}

// GetEmbedding retrieves a specific embedding.
func (p *PGVectorEmbeddings) GetEmbedding(ctx context.Context, id string) (*VectorResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	entry, ok := p.entries[id]
	if !ok {
		return nil, nil
	}
	return &VectorResult{
		ID:       entry.ID,
		Score:    entry.Score,
		Metadata: cloneMetadata(entry.Metadata),
		Vector:   append([]float32(nil), entry.Vector...),
	}, nil
}

// GetSimilarServices retrieves similar services by service ID.
func (p *PGVectorEmbeddings) GetSimilarServices(ctx context.Context, serviceID string, topK int) ([]VectorResult, error) {
	serviceResult, err := p.GetEmbedding(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	if serviceResult == nil {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}
	return p.SearchSimilar(ctx, serviceResult.Vector, topK, nil)
}

// BulkInsertEmbeddings inserts multiple embeddings.
func (p *PGVectorEmbeddings) BulkInsertEmbeddings(ctx context.Context, embeddings []*ServiceEmbedding) error {
	for _, embedding := range embeddings {
		_ = p.StoreEmbedding(ctx, embedding.ID, embedding.Vector, embeddingToMetadata(embedding))
	}
	return nil
}

// GetEmbeddingCount returns the total number of embeddings.
func (p *PGVectorEmbeddings) GetEmbeddingCount(ctx context.Context) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.entries), nil
}

// RebuildIndex is a no-op for the in-memory store.
func (p *PGVectorEmbeddings) RebuildIndex(ctx context.Context) error { return nil }

// CleanupStaleEmbeddings is a no-op for the in-memory store.
func (p *PGVectorEmbeddings) CleanupStaleEmbeddings(ctx context.Context, maxAge time.Duration) (int64, error) {
	return 0, nil
}

func cloneMetadata(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func matchesEmbeddingFilter(entry *VectorResult, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}
	for key, value := range filters {
		switch key {
		case "service_id", "category":
			if fmt.Sprint(entry.Metadata[key]) != fmt.Sprint(value) {
				return false
			}
		case "tags":
			if !strings.Contains(strings.ToLower(fmt.Sprint(entry.Metadata["tags"])), strings.ToLower(fmt.Sprint(value))) {
				return false
			}
		}
	}
	return true
}
