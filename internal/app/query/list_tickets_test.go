package query_test

import (
	"context"
	"errors"
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

func TestListTicketsService_Execute(t *testing.T) {
	t.Parallel()

	principal := entity.Principal{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Role:     "agent",
	}

	tests := []struct {
		name      string
		setupMock func(*mockservice.MockAuthorizer, *mockrepository.MockTicketRepository, *mockservice.MockAuditLogger)
		wantLen   int
		wantErr   error
	}{
		{
			name: "success returns tickets",
			setupMock: func(authorizer *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, auditLogger *mockservice.MockAuditLogger) {
				tickets := []entity.Ticket{{ID: uuid.New(), TenantID: principal.TenantID, Subject: "foo", CreatedByUserID: principal.UserID}}
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketRepo.EXPECT().ListByTenant(mock.Anything, principal.TenantID).Return(tickets, nil)
				auditLogger.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil)
			},
			wantLen: 1,
		},
		{
			name: "forbidden when not authorized",
			setupMock: func(authorizer *mockservice.MockAuthorizer, _ *mockrepository.MockTicketRepository, _ *mockservice.MockAuditLogger) {
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(service.ErrPermissionDenied)
			},
			wantErr: apperrors.ErrForbidden,
		},
		{
			name: "authorization error propagates",
			setupMock: func(authorizer *mockservice.MockAuthorizer, _ *mockrepository.MockTicketRepository, _ *mockservice.MockAuditLogger) {
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(errors.New("authz unavailable"))
			},
			wantErr: errors.New("authz unavailable"),
		},
		{
			name: "repository error propagates",
			setupMock: func(authorizer *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, _ *mockservice.MockAuditLogger) {
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketRepo.EXPECT().ListByTenant(mock.Anything, principal.TenantID).Return(nil, errors.New("db down"))
			},
			wantErr: errors.New("db down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := mockservice.NewMockAuthorizer(t)
			ticketRepo := mockrepository.NewMockTicketRepository(t)
			auditLogger := mockservice.NewMockAuditLogger(t)

			tt.setupMock(authorizer, ticketRepo, auditLogger)

			svc := query.NewListTicketsService(ticketRepo, authorizer, auditLogger)
			tickets, err := svc.Execute(context.Background(), query.ListTicketsQuery{Principal: principal})

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, apperrors.ErrForbidden) {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.EqualError(t, err, tt.wantErr.Error())
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, tickets, tt.wantLen)
		})
	}
}
