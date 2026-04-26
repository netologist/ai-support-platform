package query_test

import (
	"context"
	"testing"
	"time"

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

func TestListDocumentsService_Execute(t *testing.T) {
	principal := entity.Principal{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Email:    "agent@test.com",
		Role:     "agent",
	}
	now := time.Now().UTC()

	tests := []struct {
		name      string
		setupMock func(*mockservice.MockAuthorizer, *mockrepository.MockDocumentRepository, *mockservice.MockAuditLogger)
		wantLen   int
		wantErr   error
	}{
		{
			name: "success returns tenant documents",
			setupMock: func(authorizer *mockservice.MockAuthorizer, documentRepo *mockrepository.MockDocumentRepository, auditLogger *mockservice.MockAuditLogger) {
				authorizer.EXPECT().Authorize(mock.Anything, principal, "documents", "read").Return(nil)
				documentRepo.EXPECT().ListDocumentsByTenant(mock.Anything, principal.TenantID).Return([]entity.KnowledgeDocument{{
					ID:              uuid.New(),
					TenantID:        principal.TenantID,
					Title:           "FAQ",
					SourceURI:       "https://example.com/faq",
					CreatedByUserID: principal.UserID,
					CreatedAt:       now,
				}}, nil)
				auditLogger.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil)
			},
			wantLen: 1,
		},
		{
			name: "forbidden when not authorized",
			setupMock: func(authorizer *mockservice.MockAuthorizer, documentRepo *mockrepository.MockDocumentRepository, auditLogger *mockservice.MockAuditLogger) {
				authorizer.EXPECT().Authorize(mock.Anything, principal, "documents", "read").Return(service.ErrPermissionDenied)
			},
			wantErr: apperrors.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := mockservice.NewMockAuthorizer(t)
			documentRepo := mockrepository.NewMockDocumentRepository(t)
			auditLogger := mockservice.NewMockAuditLogger(t)

			tt.setupMock(authorizer, documentRepo, auditLogger)

			svc := query.NewListDocumentsService(documentRepo, authorizer, auditLogger)
			documents, err := svc.Execute(context.Background(), query.ListDocumentsQuery{Principal: principal})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Len(t, documents, tt.wantLen)
		})
	}
}
