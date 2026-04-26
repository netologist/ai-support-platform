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
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	mockrepository "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mockservice "github.com/netologist/ai-support-platform/internal/mocks/service"
)

func TestSuggestReplyService_Execute(t *testing.T) {
	tenantID := uuid.New()
	principal := entity.Principal{
		UserID:   uuid.New(),
		TenantID: tenantID,
		Email:    "agent@test.com",
		Role:     "agent",
	}
	ticketID := uuid.New()

	tests := []struct {
		name      string
		query     query.SuggestReplyQuery
		setupMock func(*mockservice.MockAuthorizer, *mockrepository.MockTicketRepository, *mockrepository.MockChunkRepository, *mockservice.MockEmbeddingProvider, *mockservice.MockAIProvider)
		wantReply string
		wantErr   error
	}{
		{
			name:  "success generates reply with context",
			query: query.SuggestReplyQuery{Principal: principal, TicketID: ticketID},
			setupMock: func(auth *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, chunkRepo *mockrepository.MockChunkRepository, embedder *mockservice.MockEmbeddingProvider, ai *mockservice.MockAIProvider) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketRepo.EXPECT().GetByID(mock.Anything, ticketID).
					Return(entity.Ticket{ID: ticketID, TenantID: tenantID, Subject: "Cannot login"}, nil)
				embedder.EXPECT().Embed(mock.Anything, "Cannot login").Return([]float32{0.1, 0.2}, nil)
				chunkRepo.EXPECT().SearchSimilar(mock.Anything, tenantID, []float32{0.1, 0.2}, 3).
					Return([]entity.DocumentChunk{{Content: "Try resetting your password"}}, nil)
				ai.EXPECT().GenerateReply(mock.Anything, mock.AnythingOfType("string"), "Cannot login").
					Return("Please try resetting your password.", nil)
			},
			wantReply: "Please try resetting your password.",
		},
		{
			name:  "forbidden when not authorized",
			query: query.SuggestReplyQuery{Principal: principal, TicketID: ticketID},
			setupMock: func(auth *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, chunkRepo *mockrepository.MockChunkRepository, embedder *mockservice.MockEmbeddingProvider, ai *mockservice.MockAIProvider) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(service.ErrPermissionDenied)
			},
			wantErr: apperrors.ErrForbidden,
		},
		{
			name:  "not found when ticket missing",
			query: query.SuggestReplyQuery{Principal: principal, TicketID: ticketID},
			setupMock: func(auth *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, chunkRepo *mockrepository.MockChunkRepository, embedder *mockservice.MockEmbeddingProvider, ai *mockservice.MockAIProvider) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketRepo.EXPECT().GetByID(mock.Anything, ticketID).Return(entity.Ticket{}, repository.ErrNotFound)
			},
			wantErr: apperrors.ErrNotFound,
		},
		{
			name:  "forbidden when ticket belongs to different tenant",
			query: query.SuggestReplyQuery{Principal: principal, TicketID: ticketID},
			setupMock: func(auth *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, chunkRepo *mockrepository.MockChunkRepository, embedder *mockservice.MockEmbeddingProvider, ai *mockservice.MockAIProvider) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketRepo.EXPECT().GetByID(mock.Anything, ticketID).
					Return(entity.Ticket{ID: ticketID, TenantID: uuid.New(), Subject: "test"}, nil)
			},
			wantErr: apperrors.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := mockservice.NewMockAuthorizer(t)
			ticketRepo := mockrepository.NewMockTicketRepository(t)
			chunkRepo := mockrepository.NewMockChunkRepository(t)
			embedder := mockservice.NewMockEmbeddingProvider(t)
			aiProvider := mockservice.NewMockAIProvider(t)

			tt.setupMock(authorizer, ticketRepo, chunkRepo, embedder, aiProvider)

			svc := query.NewSuggestReplyService(ticketRepo, chunkRepo, embedder, aiProvider, authorizer)
			result, err := svc.Execute(context.Background(), tt.query)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantReply, result.SuggestedReply)
			assert.NotEmpty(t, result.Sources)
		})
	}
}
