package command_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/app/command"
	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	mockrepository "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mockservice "github.com/netologist/ai-support-platform/internal/mocks/service"
)

func TestIngestDocumentService_Execute(t *testing.T) {
	principal := entity.Principal{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Email:    "admin@test.com",
		Role:     "admin",
	}

	tests := []struct {
		name      string
		cmd       command.IngestDocumentCommand
		setupMock func(*mockservice.MockAuthorizer, *mockrepository.MockDocumentRepository, *mockrepository.MockChunkRepository, *mockservice.MockChunker, *mockservice.MockEmbeddingProvider, *mockservice.MockAuditLogger)
		wantErr   error
	}{
		{
			name: "success ingests document with chunks",
			cmd: command.IngestDocumentCommand{
				Principal: principal,
				Title:     "FAQ",
				SourceURI: "https://example.com/faq",
				Content:   "First paragraph.\n\nSecond paragraph.",
			},
			setupMock: func(auth *mockservice.MockAuthorizer, docRepo *mockrepository.MockDocumentRepository, chunkRepo *mockrepository.MockChunkRepository, chunker *mockservice.MockChunker, embedder *mockservice.MockEmbeddingProvider, audit *mockservice.MockAuditLogger) {
				auth.EXPECT().Authorize(mock.Anything, principal, "documents", "create").Return(nil)
				docRepo.EXPECT().CreateDocument(mock.Anything, mock.AnythingOfType("entity.KnowledgeDocument")).
					Return(entity.KnowledgeDocument{ID: uuid.New(), TenantID: principal.TenantID, Title: "FAQ"}, nil)
				chunker.EXPECT().Chunk(mock.Anything, "First paragraph.\n\nSecond paragraph.", 512).
					Return([]string{"First paragraph.", "Second paragraph."}, nil)
				embedder.EXPECT().Embed(mock.Anything, "First paragraph.").Return(make([]float32, 1536), nil)
				embedder.EXPECT().Embed(mock.Anything, "Second paragraph.").Return(make([]float32, 1536), nil)
				embedder.EXPECT().ModelName().Return("fake-model")
				chunkRepo.EXPECT().InsertChunks(mock.Anything, mock.AnythingOfType("[]entity.DocumentChunk")).Return(nil)
				audit.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil)
			},
		},
		{
			name: "forbidden when not authorized",
			cmd: command.IngestDocumentCommand{
				Principal: principal,
				Title:     "FAQ",
				Content:   "Content",
			},
			setupMock: func(auth *mockservice.MockAuthorizer, docRepo *mockrepository.MockDocumentRepository, chunkRepo *mockrepository.MockChunkRepository, chunker *mockservice.MockChunker, embedder *mockservice.MockEmbeddingProvider, audit *mockservice.MockAuditLogger) {
				auth.EXPECT().Authorize(mock.Anything, principal, "documents", "create").Return(service.ErrPermissionDenied)
			},
			wantErr: apperrors.ErrForbidden,
		},
		{
			name: "compensating delete when InsertChunks fails",
			cmd: command.IngestDocumentCommand{
				Principal: principal,
				Title:     "FAQ",
				Content:   "Paragraph one.",
			},
			setupMock: func(auth *mockservice.MockAuthorizer, docRepo *mockrepository.MockDocumentRepository, chunkRepo *mockrepository.MockChunkRepository, chunker *mockservice.MockChunker, embedder *mockservice.MockEmbeddingProvider, audit *mockservice.MockAuditLogger) {
				createdDoc := entity.KnowledgeDocument{ID: uuid.New(), TenantID: principal.TenantID, Title: "FAQ"}
				auth.EXPECT().Authorize(mock.Anything, principal, "documents", "create").Return(nil)
				docRepo.EXPECT().CreateDocument(mock.Anything, mock.AnythingOfType("entity.KnowledgeDocument")).Return(createdDoc, nil)
				chunker.EXPECT().Chunk(mock.Anything, "Paragraph one.", 512).Return([]string{"Paragraph one."}, nil)
				embedder.EXPECT().Embed(mock.Anything, "Paragraph one.").Return(make([]float32, 1536), nil)
				embedder.EXPECT().ModelName().Return("fake-model")
				chunkRepo.EXPECT().InsertChunks(mock.Anything, mock.AnythingOfType("[]entity.DocumentChunk")).Return(assert.AnError)
				docRepo.EXPECT().DeleteDocument(mock.Anything, createdDoc.ID).Return(nil)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := mockservice.NewMockAuthorizer(t)
			docRepo := mockrepository.NewMockDocumentRepository(t)
			chunkRepo := mockrepository.NewMockChunkRepository(t)
			chunker := mockservice.NewMockChunker(t)
			embedder := mockservice.NewMockEmbeddingProvider(t)
			auditLogger := mockservice.NewMockAuditLogger(t)

			tt.setupMock(authorizer, docRepo, chunkRepo, chunker, embedder, auditLogger)

			svc := command.NewIngestDocumentService(docRepo, chunkRepo, chunker, embedder, authorizer, auditLogger)
			doc, err := svc.Execute(context.Background(), tt.cmd)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "FAQ", doc.Title)
		})
	}
}
