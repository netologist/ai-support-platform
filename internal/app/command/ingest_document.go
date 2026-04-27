package command

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type IngestDocumentCommand struct {
	Principal entity.Principal
	Title     string
	SourceURI string
	Content   string
}

type IngestDocumentService struct {
	documentRepo repository.DocumentRepository
	chunkRepo    repository.ChunkRepository
	chunker      service.Chunker
	embedder     service.EmbeddingProvider
	authorizer   service.Authorizer
	auditLogger  service.AuditLogger
}

func NewIngestDocumentService(
	documentRepo repository.DocumentRepository,
	chunkRepo repository.ChunkRepository,
	chunker service.Chunker,
	embedder service.EmbeddingProvider,
	authorizer service.Authorizer,
	auditLogger service.AuditLogger,
) IngestDocumentService {
	return IngestDocumentService{
		documentRepo: documentRepo,
		chunkRepo:    chunkRepo,
		chunker:      chunker,
		embedder:     embedder,
		authorizer:   authorizer,
		auditLogger:  auditLogger,
	}
}

func (svc IngestDocumentService) Execute(ctx context.Context, cmd IngestDocumentCommand) (entity.KnowledgeDocument, error) {
	if err := svc.authorizer.Authorize(ctx, cmd.Principal, "documents", "create"); err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			return entity.KnowledgeDocument{}, apperrors.ErrForbidden
		}
		return entity.KnowledgeDocument{}, err
	}

	now := time.Now().UTC()
	doc := entity.KnowledgeDocument{
		ID:              uuid.New(),
		TenantID:        cmd.Principal.TenantID,
		Title:           cmd.Title,
		SourceURI:       cmd.SourceURI,
		CreatedByUserID: cmd.Principal.UserID,
		CreatedAt:       now,
	}

	createdDoc, err := svc.documentRepo.CreateDocument(ctx, doc)
	if err != nil {
		return entity.KnowledgeDocument{}, err
	}

	textChunks, err := svc.chunker.Chunk(ctx, cmd.Content, 512)
	if err != nil {
		return entity.KnowledgeDocument{}, err
	}

	domainChunks := make([]entity.DocumentChunk, 0, len(textChunks))
	for i, text := range textChunks {
		embedding, err := svc.embedder.Embed(ctx, text)
		if err != nil {
			return entity.KnowledgeDocument{}, err
		}

		domainChunks = append(domainChunks, entity.DocumentChunk{
			ID:             uuid.New(),
			DocumentID:     createdDoc.ID,
			TenantID:       createdDoc.TenantID,
			ChunkIndex:     i,
			Content:        text,
			EmbeddingModel: svc.embedder.ModelName(),
			Embedding:      embedding,
			CreatedAt:      now,
		})
	}

	if err := svc.chunkRepo.InsertChunks(ctx, domainChunks); err != nil {
		// Compensating action: delete document to maintain consistency
		if delErr := svc.documentRepo.DeleteDocument(ctx, createdDoc.ID); delErr != nil {
			slog.Error("failed to delete document after chunk insert failure",
				slog.String("document_id", createdDoc.ID.String()),
				slog.Any("delete_error", delErr),
				slog.Any("insert_error", err),
			)
		}
		return entity.KnowledgeDocument{}, err
	}

	if svc.auditLogger != nil {
		_ = svc.auditLogger.Record(ctx, entity.AuditLog{
			EventType:  "document.ingested",
			Action:     "create",
			Outcome:    "success",
			TenantID:   &cmd.Principal.TenantID,
			UserID:     &cmd.Principal.UserID,
			Resource:   "document",
			ResourceID: createdDoc.ID.String(),
			Metadata:   map[string]any{"chunks": len(domainChunks)},
		})
	}

	return createdDoc, nil
}
