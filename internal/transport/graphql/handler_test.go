package graphql

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	mockrepository "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mockservice "github.com/netologist/ai-support-platform/internal/mocks/service"
	"github.com/netologist/ai-support-platform/internal/transport"
)

func TestHandler_ServeHTTP_TicketsAndDocuments(t *testing.T) {
	principal := entity.Principal{UserID: uuid.New(), TenantID: uuid.New(), Email: "agent@test.com", Role: "agent"}
	createdAt := time.Now().UTC()

	authorizer := mockservice.NewMockAuthorizer(t)
	ticketRepo := mockrepository.NewMockTicketRepository(t)
	documentRepo := mockrepository.NewMockDocumentRepository(t)
	auditLogger := mockservice.NewMockAuditLogger(t)

	authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
	ticketRepo.EXPECT().ListByTenant(mock.Anything, principal.TenantID).Return([]entity.Ticket{{
		ID:              uuid.New(),
		TenantID:        principal.TenantID,
		Subject:         "Cannot login",
		Status:          "open",
		CreatedByUserID: principal.UserID,
		CreatedAt:       createdAt,
	}}, nil)
	auditLogger.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil).Once()

	authorizer.EXPECT().Authorize(mock.Anything, principal, "documents", "read").Return(nil)
	documentRepo.EXPECT().ListDocumentsByTenant(mock.Anything, principal.TenantID).Return([]entity.KnowledgeDocument{{
		ID:              uuid.New(),
		TenantID:        principal.TenantID,
		Title:           "FAQ",
		SourceURI:       "https://example.com/faq",
		CreatedByUserID: principal.UserID,
		CreatedAt:       createdAt,
	}}, nil)
	auditLogger.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil).Once()

	handler := NewHandler(Dependencies{
		ListTicketsService:   query.NewListTicketsService(ticketRepo, authorizer, auditLogger),
		ListDocumentsService: query.NewListDocumentsService(documentRepo, authorizer, auditLogger),
	})

	body := bytes.NewBufferString(`{"query":"query Dashboard { tickets { id subject } documents { id title } }","variables":{}}`)
	request := httptest.NewRequest(http.MethodPost, "/graphql", body)
	request = request.WithContext(transport.WithPrincipal(request.Context(), principal))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "Cannot login")
	require.Contains(t, response.Body.String(), "FAQ")
}

func TestHandler_ServeHTTP_SearchDocumentsAndSuggestReply(t *testing.T) {
	principal := entity.Principal{UserID: uuid.New(), TenantID: uuid.New(), Email: "agent@test.com", Role: "agent"}
	ticketID := uuid.New()

	authorizer := mockservice.NewMockAuthorizer(t)
	ticketRepo := mockrepository.NewMockTicketRepository(t)
	chunkRepo := mockrepository.NewMockChunkRepository(t)
	embedder := mockservice.NewMockEmbeddingProvider(t)
	aiProvider := mockservice.NewMockAIProvider(t)

	authorizer.EXPECT().Authorize(mock.Anything, principal, "documents", "read").Return(nil)
	embedder.EXPECT().Embed(mock.Anything, "password reset").Return([]float32{0.1, 0.2}, nil)
	chunkRepo.EXPECT().SearchSimilar(mock.Anything, principal.TenantID, []float32{0.1, 0.2}, 2).Return([]entity.DocumentChunk{{
		ID:             uuid.New(),
		DocumentID:     uuid.New(),
		TenantID:       principal.TenantID,
		ChunkIndex:     0,
		Content:        "Reset password instructions",
		EmbeddingModel: "fake-model",
	}}, nil)

	authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
	ticketRepo.EXPECT().GetByID(mock.Anything, ticketID).Return(entity.Ticket{
		ID:              ticketID,
		TenantID:        principal.TenantID,
		Subject:         "Cannot login",
		Status:          "open",
		CreatedByUserID: principal.UserID,
	}, nil)
	embedder.EXPECT().Embed(mock.Anything, "Cannot login").Return([]float32{0.3, 0.4}, nil)
	chunkRepo.EXPECT().SearchSimilar(mock.Anything, principal.TenantID, []float32{0.3, 0.4}, 3).Return([]entity.DocumentChunk{{
		ID:             uuid.New(),
		DocumentID:     uuid.New(),
		TenantID:       principal.TenantID,
		ChunkIndex:     0,
		Content:        "Reset password instructions",
		EmbeddingModel: "fake-model",
	}}, nil)
	aiProvider.EXPECT().GenerateReply(mock.Anything, mock.AnythingOfType("string"), "Cannot login").Return("Try resetting your password.", nil)

	handler := NewHandler(Dependencies{
		SemanticSearchService: query.NewSemanticSearchService(chunkRepo, embedder, authorizer),
		SuggestReplyService:   query.NewSuggestReplyService(ticketRepo, chunkRepo, embedder, aiProvider, authorizer),
	})

	body := bytes.NewBufferString(`{"query":"query Search($query: String!, $limit: Int!, $ticketId: ID!) { searchDocuments(query: $query, limit: $limit) { id } suggestReply(ticketId: $ticketId) { suggestedReply } }","variables":{"query":"password reset","limit":2,"ticketId":"` + ticketID.String() + `"}}`)
	request := httptest.NewRequest(http.MethodPost, "/graphql", body)
	request = request.WithContext(transport.WithPrincipal(request.Context(), principal))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "Reset password instructions")
	require.Contains(t, response.Body.String(), "Try resetting your password.")
}

func TestHandler_ServeHTTP_RequiresPrincipal(t *testing.T) {
	handler := NewHandler(Dependencies{})
	request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(`{"query":"query { tickets { id } }"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.NotNil(t, payload["errors"])
}
