package entity

import (
	"time"

	"github.com/google/uuid"
)

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
