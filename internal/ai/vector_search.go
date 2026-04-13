package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// PGVectorEmbeddings implements vector search using PostgreSQL pgvector
type PGVectorEmbeddings struct {
	logger *logrus.Logger
	db     *sql.DB
	config PGVectorConfig
}

// PGVectorConfig contains configuration for pgvector
type PGVectorConfig struct {
	CollectionSize int
	COSIMDistance  bool
	TopK           int
	Threshold      float32
}

// NewPGVectorEmbeddings creates a new pgvector embeddings client
func NewPGVectorEmbeddings(logger *logrus.Logger, db *sql.DB, config PGVectorConfig) *PGVectorEmbeddings {
	return &PGVectorEmbeddings{
		logger: logger.WithField("component", "pgvector"),
		db:     db,
		config: config,
	}
}

// Initialize creates the necessary database tables and indexes
func (p *PGVectorEmbeddings) Initialize(ctx context.Context) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS service_embeddings (
		id TEXT PRIMARY KEY,
		service_id TEXT NOT NULL,
		service_name TEXT NOT NULL,
		category TEXT,
		tags TEXT[],
		metadata JSONB,
		embedding VECTOR(1536),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := p.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		p.logger.WithError(err).Error("Failed to create embeddings table")
		return err
	}

	// Create IVFFLAT index for efficient vector search
	createIndexSQL := `
	CREATE INDEX IF NOT EXISTS service_embeddings_vector_idx
	ON service_embeddings
	USING ivfflat (embedding vector_cosine_similarity)
	WITH (lists = 100);
	`

	_, err = p.db.ExecContext(ctx, createIndexSQL)
	if err != nil {
		p.logger.WithError(err).Error("Failed to create vector index")
		return err
	}

	// Create GIN index for metadata filtering
	createMetaIndexSQL := `
	CREATE INDEX IF NOT EXISTS service_embeddings_metadata_idx
	ON service_embeddings USING GIN (metadata jsonb);
	`

	_, err = p.db.ExecContext(ctx, createMetaIndexSQL)
	if err != nil {
		p.logger.WithError(err).Error("Failed to create metadata index")
		return err
	}

	p.logger.Info("PGVector embeddings initialized successfully")
	return nil
}

// StoreEmbedding stores a service embedding in the database
func (p *PGVectorEmbeddings) StoreEmbedding(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	// Convert vector to pgvector format
	vectorStr := vectorToPGVector(vector)

	// Convert metadata to JSON
	metadataJSON, err := p.convertMetadataToJSON(metadata)
	if err != nil {
		return err
	}

	// Get service info from metadata
	serviceID, _ := metadata["service_id"].(string)
	serviceName, _ := metadata["service_name"].(string)
	category, _ := metadata["category"].(string)

	// Convert tags to array
	tags := p.convertTagsToPGArray(metadata["tags])

	insertSQL := `
	INSERT INTO service_embeddings (id, service_id, service_name, category, tags, metadata, embedding)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (id) DO UPDATE SET
		service_id = EXCLUDED.service_id,
		service_name = EXCLUDED.service_name,
		category = EXCLUDED.category,
		tags = EXCLUDED.tags,
		metadata = EXCLUDED.metadata,
		embedding = EXCLUDED.embedding,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err = p.db.ExecContext(ctx, insertSQL,
		id, serviceID, serviceName, category, tags, metadataJSON, vectorStr)

	if err != nil {
		p.logger.WithError(err).Error("Failed to store embedding")
		return err
	}

	p.logger.WithFields(logrus.Fields{
		"id":         id,
		"service_id": serviceID,
		"vector_dim": len(vector),
	}).Debug("Embedding stored")

	return nil
}

// SearchSimilar performs a similarity search
func (p *PGVectorEmbeddings) SearchSimilar(ctx context.Context, queryVector []float32, topK int, filters map[string]interface{}) ([]VectorResult, error) {
	vectorStr := vectorToPGVector(queryVector)

	// Build base SQL
	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString("SELECT id, service_id, service_name, category, tags, metadata, ")
	sqlBuilder.WriteString("1 - (embedding <=> $1::vector) AS similarity_score ")
	sqlBuilder.WriteString("FROM service_embeddings WHERE ")

	// Add filters
	var conditions []string
	var args []interface{}
	args = append(args, vectorStr)

	if len(filters) > 0 {
		if serviceID, ok := filters["service_id"]; ok {
			conditions = append(conditions, "service_id = $"+fmt.Sprintf("%d", len(args)+1))
			args = append(args, serviceID)
		}
		if category, ok := filters["category"]; ok {
			conditions = append(conditions, "category = $"+fmt.Sprintf("%d", len(args)+1))
			args = append(args, category)
		}
		if tags, ok := filters["tags"]; ok {
			tagsArray := p.convertTagsToPGArray(tags)
			conditions = append(conditions, "tags && $"+fmt.Sprintf("%d", len(args)+1))
			args = append(args, tagsArray)
		}
		if metadata, ok := filters["metadata"]; ok {
			metadataJSON, _ := p.convertMetadataToJSON(metadata)
			conditions = append(conditions, "metadata @> $"+fmt.Sprintf("%d", len(args)+1))
			args = append(args, metadataJSON)
		}
	}

	if len(conditions) > 0 {
		sqlBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	// Add ordering and limiting
	sqlBuilder.WriteString(" ORDER BY similarity_score DESC LIMIT $1")
	args = append(args, topK)

	// Execute query
	rows, err := p.db.QueryContext(ctx, sqlBuilder.String(), args...)
	if err != nil {
		p.logger.WithError(err).Error("Failed to search embeddings")
		return nil, err
	}
	defer rows.Close()

	var results []VectorResult
	for rows.Next() {
		var result VectorResult
		var similarityScore float32
		var metadataJSON []byte
		var tagsString string

		err := rows.Scan(
			&result.ID,
			&result.Metadata["service_id"],
			&result.Metadata["service_name"],
			&result.Metadata["category"],
			&tagsString,
			&metadataJSON,
			&similarityScore,
		)
		if err != nil {
			p.logger.WithError(err).Error("Failed to scan result")
			continue
		}

		result.Score = similarityScore
		result.Metadata["id"] = result.ID

		// Parse tags
		tags := parsePGArray(tagsString)
		result.Metadata["tags"] = tags

		// Parse JSON metadata
		var metaMap map[string]interface{}
		if err := json.Unmarshal(metadataJSON, &metaMap); err == nil {
			// Merge with existing metadata
			for k, v := range metaMap {
				result.Metadata[k] = v
			}
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		p.logger.WithError(err).Error("Error iterating over results")
		return nil, err
	}

	p.logger.WithFields(logrus.Fields{
		"top_k":    topK,
		"results":  len(results),
		"filters":  len(filters),
	}).Debug("Search completed")

	return results, nil
}

// DeleteEmbedding removes an embedding from the database
func (p *PGVectorEmbeddings) DeleteEmbedding(ctx context.Context, id string) error {
	deleteSQL := `DELETE FROM service_embeddings WHERE id = $1`
	_, err := p.db.ExecContext(ctx, deleteSQL, id)
	if err != nil {
		p.logger.WithError(err).Error("Failed to delete embedding")
		return err
	}

	p.logger.WithField("id", id).Debug("Embedding deleted")
	return nil
}

// GetEmbedding retrieves a specific embedding
func (p *PGVectorEmbeddings) GetEmbedding(ctx context.Context, id string) (*VectorResult, error) {
	selectSQL := `
	SELECT id, service_id, service_name, category, tags, metadata, embedding
	FROM service_embeddings WHERE id = $1
	`

	var result VectorResult
	var serviceID, serviceName, category, tagsString string
	var metadataJSON []byte
	var embeddingBytes []byte

	err := p.db.QueryRowContext(ctx, selectSQL, id).Scan(
		&result.ID,
		&serviceID,
		&serviceName,
		&category,
		&tagsString,
		&metadataJSON,
		&embeddingBytes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		p.logger.WithError(err).Error("Failed to get embedding")
		return nil, err
	}

	result.Metadata = make(map[string]interface{})
	result.Metadata["service_id"] = serviceID
	result.Metadata["service_name"] = serviceName
	result.Metadata["category"] = category
	result.Metadata["tags"] = parsePGArray(tagsString)

	var metaMap map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metaMap); err == nil {
		for k, v := range metaMap {
			result.Metadata[k] = v
		}
	}

	// Convert vector bytes back to slice
	result.Vector = pgVectorToFloat32(embeddingBytes)

	return &result, nil
}

// GetSimilarServices retrieves similar services by service ID
func (p *PGVectorEmbeddings) GetSimilarServices(ctx context.Context, serviceID string, topK int) ([]VectorResult, error) {
	// First get the service's embedding
	serviceResult, err := p.getServiceEmbeddingByServiceID(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	if serviceResult == nil {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	// Search for similar services
	return p.SearchSimilar(ctx, serviceResult.Vector, topK, nil)
}

// getServiceEmbeddingByServiceID retrieves embedding by service ID
func (p *PGVectorEmbeddings) getServiceEmbeddingByServiceID(ctx context.Context, serviceID string) (*VectorResult, error) {
	selectSQL := `
	SELECT id, service_name, category, tags, metadata, embedding
	FROM service_embeddings WHERE service_id = $1 LIMIT 1
	`

	var result VectorResult
	var serviceName, category, tagsString string
	var metadataJSON []byte
	var embeddingBytes []byte

	err := p.db.QueryRowContext(ctx, selectSQL, serviceID).Scan(
		&result.ID,
		&serviceName,
		&category,
		&tagsString,
		&metadataJSON,
		&embeddingBytes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	result.Metadata = make(map[string]interface{})
	result.Metadata["service_id"] = serviceID
	result.Metadata["service_name"] = serviceName
	result.Metadata["category"] = category
	result.Metadata["tags"] = parsePGArray(tagsString)

	var metaMap map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metaMap); err == nil {
		for k, v := range metaMap {
			result.Metadata[k] = v
		}
	}

	result.Vector = pgVectorToFloat32(embeddingBytes)

	return &result, nil
}

// BulkInsertEmbeddings inserts multiple embeddings at once
func (p *PGVectorEmbeddings) BulkInsertEmbeddings(ctx context.Context, embeddings []*ServiceEmbedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertSQL := `
	INSERT INTO service_embeddings (id, service_id, service_name, category, tags, metadata, embedding)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (id) DO UPDATE SET
		service_id = EXCLUDED.service_id,
		service_name = EXCLUDED.service_name,
		category = EXCLUDED.category,
		tags = EXCLUDED.tags,
		metadata = EXCLUDED.metadata,
		embedding = EXCLUDED.embedding,
		updated_at = CURRENT_TIMESTAMP
	`

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, embedding := range embeddings {
		vectorStr := vectorToPGVector(embedding.Vector)
		metadataJSON, _ := p.convertMetadataToJSON(embedding.Metadata)
		tags := p.convertTagsToPGArray(embedding.Tags)

		_, err := stmt.ExecContext(ctx,
			embedding.ID,
			embedding.ServiceID,
			embedding.ServiceName,
			embedding.Category,
			tags,
			metadataJSON,
			vectorStr,
		)
		if err != nil {
			p.logger.WithError(err).Error("Failed to insert embedding")
			return err
		}
	}

	return tx.Commit()
}

// GetEmbeddingCount returns the total number of embeddings
func (p *PGVectorEmbeddings) GetEmbeddingCount(ctx context.Context) (int, error) {
	var count int
	err := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_embeddings").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// RebuildIndex rebuilds the vector index for better performance
func (p *PGVectorEmbeddings) RebuildIndex(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, "REINDEX INDEX service_embeddings_vector_idx")
	if err != nil {
		p.logger.WithError(err).Error("Failed to rebuild index")
		return err
	}

	p.logger.Info("Vector index rebuilt successfully")
	return nil
}

// CleanupStaleEmbeddings removes embeddings older than the specified duration
func (p *PGVectorEmbeddings) CleanupStaleEmbeddings(ctx context.Context, maxAge time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-maxAge)
	result, err := p.db.ExecContext(ctx,
		"DELETE FROM service_embeddings WHERE updated_at < $1",
		cutoffTime,
	)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Helper functions

// vectorToPGVector converts a float32 slice to pgvector string format
func vectorToPGVector(vector []float32) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range vector {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%g", v))
	}
	sb.WriteString("]")
	return sb.String()
}

// pgVectorToFloat32 converts pgvector string format to float32 slice
func pgVectorToFloat32(data []byte) []float32 {
	// This is a simplified version - in production would properly parse the vector format
	var vector []float32
	// Actual implementation depends on pgvector format
	// For now, return empty
	return vector
}

// convertMetadataToJSON converts metadata map to JSON
func (p *PGVectorEmbeddings) convertMetadataToJSON(metadata map[string]interface{}) ([]byte, error) {
	return json.Marshal(metadata)
}

// convertTagsToPGArray converts tags slice to PostgreSQL array string
func (p *PGVectorEmbeddings) convertTagsToPGArray(tags interface{}) string {
	tagSlice, ok := tags.([]string)
	if !ok {
		return "{}"
	}

	var sb strings.Builder
	sb.WriteString("{")
	for i, tag := range tagSlice {
		if i > 0 {
			sb.WriteString(",")
		}
		// Escape quotes in tag
		tag = strings.ReplaceAll(tag, "\"", "\"\"")
		sb.WriteString(fmt.Sprintf("\"%s\"", tag))
	}
	sb.WriteString("}")
	return sb.String()
}

// parsePGArray parses a PostgreSQL array string to Go slice
func parsePGArray(s string) []string {
	var result []string
	if s == "{}" || s == "" {
		return result
	}

	// Remove braces
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")

	// Split by comma
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Remove quotes
		part = strings.TrimPrefix(part, "\"")
		part = strings.TrimSuffix(part, "\"")
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

// GetSimilarityThreshold returns the current similarity threshold
func (p *PGVectorEmbeddings) GetSimilarityThreshold() float32 {
	return p.config.Threshold
}

// SetSimilarityThreshold updates the similarity threshold
func (p *PGVectorEmbeddings) SetSimilarityThreshold(threshold float32) {
	p.config.Threshold = threshold
}

// GetConfig returns the current configuration
func (p *PGVectorEmbeddings) GetConfig() PGVectorConfig {
	return p.config
}
