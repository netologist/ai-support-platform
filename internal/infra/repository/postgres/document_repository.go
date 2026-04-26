package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	domainrepository "github.com/netologist/ai-support-platform/internal/domain/repository"
	generated "github.com/netologist/ai-support-platform/internal/infra/repository/sqlc"
)

type DocumentRepository struct {
	queries *generated.Queries
}

func NewDocumentRepository(queries *generated.Queries) DocumentRepository {
	return DocumentRepository{queries: queries}
}

func (r DocumentRepository) CreateDocument(ctx context.Context, doc entity.KnowledgeDocument) (entity.KnowledgeDocument, error) {
	createdAt := doc.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	row, err := r.queries.CreateDocument(ctx, generated.CreateDocumentParams{
		ID:              toPGUUID(doc.ID),
		TenantID:        toPGUUID(doc.TenantID),
		Title:           doc.Title,
		SourceUri:       doc.SourceURI,
		CreatedByUserID: toPGUUID(doc.CreatedByUserID),
		CreatedAt:       toPGTimestamptz(createdAt),
	})
	if err != nil {
		return entity.KnowledgeDocument{}, err
	}

	return mapDocument(row)
}

func (r DocumentRepository) GetDocumentByID(ctx context.Context, id uuid.UUID) (entity.KnowledgeDocument, error) {
	row, err := r.queries.GetDocumentByID(ctx, toPGUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.KnowledgeDocument{}, domainrepository.ErrNotFound
		}
		return entity.KnowledgeDocument{}, err
	}

	return mapDocument(row)
}

func (r DocumentRepository) ListDocumentsByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.KnowledgeDocument, error) {
	rows, err := r.queries.ListDocumentsByTenant(ctx, toPGUUID(tenantID))
	if err != nil {
		return nil, err
	}

	docs := make([]entity.KnowledgeDocument, 0, len(rows))
	for _, row := range rows {
		doc, err := mapDocument(row)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

func mapDocument(row generated.KnowledgeDocument) (entity.KnowledgeDocument, error) {
	id, err := toDomainUUID(row.ID)
	if err != nil {
		return entity.KnowledgeDocument{}, err
	}
	tenantID, err := toDomainUUID(row.TenantID)
	if err != nil {
		return entity.KnowledgeDocument{}, err
	}
	createdByUserID, err := toDomainUUID(row.CreatedByUserID)
	if err != nil {
		return entity.KnowledgeDocument{}, err
	}
	t, err := toTime(row.CreatedAt)
	if err != nil {
		return entity.KnowledgeDocument{}, err
	}

	return entity.KnowledgeDocument{
		ID:              id,
		TenantID:        tenantID,
		Title:           row.Title,
		SourceURI:       row.SourceUri,
		CreatedByUserID: createdByUserID,
		CreatedAt:       t,
	}, nil
}

type ChunkRepository struct {
	queries *generated.Queries
}

func NewChunkRepository(queries *generated.Queries) ChunkRepository {
	return ChunkRepository{queries: queries}
}

func (r ChunkRepository) InsertChunks(ctx context.Context, chunks []entity.DocumentChunk) error {
	for _, chunk := range chunks {
		createdAt := chunk.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}

		err := r.queries.InsertDocumentChunk(ctx, generated.InsertDocumentChunkParams{
			ID:             toPGUUID(chunk.ID),
			DocumentID:     toPGUUID(chunk.DocumentID),
			TenantID:       toPGUUID(chunk.TenantID),
			ChunkIndex:     int32(chunk.ChunkIndex),
			Content:        chunk.Content,
			EmbeddingModel: chunk.EmbeddingModel,
			Embedding:      pgvector.NewVector(chunk.Embedding),
			CreatedAt:      toPGTimestamptz(createdAt),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r ChunkRepository) SearchSimilar(ctx context.Context, tenantID uuid.UUID, embedding []float32, limit int) ([]entity.DocumentChunk, error) {
	rows, err := r.queries.SearchSimilarChunks(ctx, generated.SearchSimilarChunksParams{
		TenantID:  toPGUUID(tenantID),
		Embedding: pgvector.NewVector(embedding),
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}

	chunks := make([]entity.DocumentChunk, 0, len(rows))
	for _, row := range rows {
		id, err := toDomainUUID(row.ID)
		if err != nil {
			return nil, err
		}
		docID, err := toDomainUUID(row.DocumentID)
		if err != nil {
			return nil, err
		}
		tid, err := toDomainUUID(row.TenantID)
		if err != nil {
			return nil, err
		}
		t, err := toTime(row.CreatedAt)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, entity.DocumentChunk{
			ID:             id,
			DocumentID:     docID,
			TenantID:       tid,
			ChunkIndex:     int(row.ChunkIndex),
			Content:        row.Content,
			EmbeddingModel: row.EmbeddingModel,
			Embedding:      row.Embedding.Slice(),
			CreatedAt:      t,
		})
	}

	return chunks, nil
}
