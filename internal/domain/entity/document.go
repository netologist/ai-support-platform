package entity

import (
	"time"

	"github.com/google/uuid"
)

// EmbeddingDimension is the vector size expected by the pgvector column
// defined in migrations/000005_documents.sql. If the embedding model changes,
// update this constant AND create a new migration to ALTER the column.
const EmbeddingDimension = 1536

type KnowledgeDocument struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Title           string
	SourceURI       string
	CreatedByUserID uuid.UUID
	CreatedAt       time.Time
}

type DocumentChunk struct {
	ID             uuid.UUID
	DocumentID     uuid.UUID
	TenantID       uuid.UUID
	ChunkIndex     int
	Content        string
	EmbeddingModel string
	Embedding      []float32
	CreatedAt      time.Time
}
