package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type DocumentRepository interface {
	CreateDocument(ctx context.Context, doc entity.KnowledgeDocument) (entity.KnowledgeDocument, error)
	GetDocumentByID(ctx context.Context, id uuid.UUID) (entity.KnowledgeDocument, error)
	ListDocumentsByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.KnowledgeDocument, error)
}

type ChunkRepository interface {
	InsertChunks(ctx context.Context, chunks []entity.DocumentChunk) error
	SearchSimilar(ctx context.Context, tenantID uuid.UUID, embedding []float32, limit int) ([]entity.DocumentChunk, error)
}
