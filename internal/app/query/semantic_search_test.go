package query_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	mockrepository "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mockservice "github.com/netologist/ai-support-platform/internal/mocks/service"
)

func TestSemanticSearchService_Execute(t *testing.T) {
	principal := entity.Principal{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Email:    "agent@test.com",
		Role:     "agent",
	}

	tests := []struct {
		name      string
		query     query.SemanticSearchQuery
		setupMock func(*mockservice.MockAuthorizer, *mockrepository.MockChunkRepository, *mockservice.MockEmbeddingProvider)
		wantLen   int
		wantErr   error
	}{
		{
			name:  "success returns matching chunks",
			query: query.SemanticSearchQuery{Principal: principal, Text: "how to reset password", Limit: 5},
			setupMock: func(auth *mockservice.MockAuthorizer, chunkRepo *mockrepository.MockChunkRepository, embedder *mockservice.MockEmbeddingProvider) {
				auth.EXPECT().Authorize(mock.Anything, principal, "documents", "read").Return(nil)
				embedder.EXPECT().Embed(mock.Anything, "how to reset password").Return([]float32{0.1, 0.2}, nil)
				chunkRepo.EXPECT().SearchSimilar(mock.Anything, principal.TenantID, []float32{0.1, 0.2}, 5).
					Return([]entity.DocumentChunk{{Content: "Reset your password by clicking..."}}, nil)
			},
			wantLen: 1,
		},
		{
			name:  "default limit when zero",
			query: query.SemanticSearchQuery{Principal: principal, Text: "test", Limit: 0},
			setupMock: func(auth *mockservice.MockAuthorizer, chunkRepo *mockrepository.MockChunkRepository, embedder *mockservice.MockEmbeddingProvider) {
				auth.EXPECT().Authorize(mock.Anything, principal, "documents", "read").Return(nil)
				embedder.EXPECT().Embed(mock.Anything, "test").Return([]float32{0.1}, nil)
				chunkRepo.EXPECT().SearchSimilar(mock.Anything, principal.TenantID, []float32{0.1}, 5).
					Return(nil, nil)
			},
			wantLen: 0,
		},
		{
			name:  "forbidden when not authorized",
			query: query.SemanticSearchQuery{Principal: principal, Text: "test"},
			setupMock: func(auth *mockservice.MockAuthorizer, chunkRepo *mockrepository.MockChunkRepository, embedder *mockservice.MockEmbeddingProvider) {
				auth.EXPECT().Authorize(mock.Anything, principal, "documents", "read").Return(service.ErrPermissionDenied)
			},
			wantErr: apperrors.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := mockservice.NewMockAuthorizer(t)
			chunkRepo := mockrepository.NewMockChunkRepository(t)
			embedder := mockservice.NewMockEmbeddingProvider(t)

			tt.setupMock(authorizer, chunkRepo, embedder)

			svc := query.NewSemanticSearchService(chunkRepo, embedder, authorizer)
			result, err := svc.Execute(context.Background(), tt.query)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Chunks, tt.wantLen)
		})
	}
}
