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
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	mockrepository "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mockservice "github.com/netologist/ai-support-platform/internal/mocks/service"
)

func TestGetTicketService_Execute(t *testing.T) {
	t.Parallel()

	principal := entity.Principal{UserID: uuid.New(), TenantID: uuid.New(), Role: "agent"}
	ticketID := uuid.New()
	queryInput := query.GetTicketQuery{
		TicketID:   ticketID,
		Principal:  principal,
		Resource:   "tickets",
		ActionName: "read",
	}

	tests := []struct {
		name      string
		setupMock func(*mockservice.MockAuthorizer, *mockrepository.MockTicketRepository, *mockservice.MockTicketCache, *mockservice.MockAuditLogger)
		assertFn  func(*testing.T, entity.Ticket, error)
	}{
		{
			name: "forbidden when not authorized",
			setupMock: func(authorizer *mockservice.MockAuthorizer, _ *mockrepository.MockTicketRepository, _ *mockservice.MockTicketCache, auditLogger *mockservice.MockAuditLogger) {
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(service.ErrPermissionDenied)
				auditLogger.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil)
			},
			assertFn: func(t *testing.T, _ entity.Ticket, err error) {
				require.ErrorIs(t, err, apperrors.ErrForbidden)
			},
		},
		{
			name: "authorization error propagates",
			setupMock: func(authorizer *mockservice.MockAuthorizer, _ *mockrepository.MockTicketRepository, _ *mockservice.MockTicketCache, _ *mockservice.MockAuditLogger) {
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(errors.New("authz unavailable"))
			},
			assertFn: func(t *testing.T, _ entity.Ticket, err error) {
				require.EqualError(t, err, "authz unavailable")
			},
		},
		{
			name: "returns cached ticket on cache hit",
			setupMock: func(authorizer *mockservice.MockAuthorizer, _ *mockrepository.MockTicketRepository, ticketCache *mockservice.MockTicketCache, auditLogger *mockservice.MockAuditLogger) {
				cached := entity.Ticket{ID: ticketID, TenantID: principal.TenantID, Subject: "cached", CreatedByUserID: principal.UserID}
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketCache.EXPECT().GetTicket(mock.Anything, ticketID).Return(cached, true, nil)
				auditLogger.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil)
			},
			assertFn: func(t *testing.T, ticket entity.Ticket, err error) {
				require.NoError(t, err)
				assert.Equal(t, "cached", ticket.Subject)
			},
		},
		{
			name: "forbidden when cache hit belongs to another tenant",
			setupMock: func(authorizer *mockservice.MockAuthorizer, _ *mockrepository.MockTicketRepository, ticketCache *mockservice.MockTicketCache, _ *mockservice.MockAuditLogger) {
				cached := entity.Ticket{ID: ticketID, TenantID: uuid.New(), Subject: "cached", CreatedByUserID: principal.UserID}
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketCache.EXPECT().GetTicket(mock.Anything, ticketID).Return(cached, true, nil)
			},
			assertFn: func(t *testing.T, _ entity.Ticket, err error) {
				require.ErrorIs(t, err, apperrors.ErrForbidden)
			},
		},
		{
			name: "returns ErrNotFound when repository does not contain ticket",
			setupMock: func(authorizer *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, ticketCache *mockservice.MockTicketCache, auditLogger *mockservice.MockAuditLogger) {
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketCache.EXPECT().GetTicket(mock.Anything, ticketID).Return(entity.Ticket{}, false, nil)
				ticketRepo.EXPECT().GetByID(mock.Anything, ticketID).Return(entity.Ticket{}, repository.ErrNotFound)
				auditLogger.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil)
			},
			assertFn: func(t *testing.T, _ entity.Ticket, err error) {
				require.ErrorIs(t, err, apperrors.ErrNotFound)
			},
		},
		{
			name: "repository error propagates",
			setupMock: func(authorizer *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, ticketCache *mockservice.MockTicketCache, _ *mockservice.MockAuditLogger) {
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketCache.EXPECT().GetTicket(mock.Anything, ticketID).Return(entity.Ticket{}, false, nil)
				ticketRepo.EXPECT().GetByID(mock.Anything, ticketID).Return(entity.Ticket{}, errors.New("db down"))
			},
			assertFn: func(t *testing.T, _ entity.Ticket, err error) {
				require.EqualError(t, err, "db down")
			},
		},
		{
			name: "forbidden when repository ticket belongs to another tenant",
			setupMock: func(authorizer *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, ticketCache *mockservice.MockTicketCache, auditLogger *mockservice.MockAuditLogger) {
				ticket := entity.Ticket{ID: ticketID, TenantID: uuid.New(), Subject: "repo", CreatedByUserID: principal.UserID}
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketCache.EXPECT().GetTicket(mock.Anything, ticketID).Return(entity.Ticket{}, false, nil)
				ticketRepo.EXPECT().GetByID(mock.Anything, ticketID).Return(ticket, nil)
				auditLogger.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil)
			},
			assertFn: func(t *testing.T, _ entity.Ticket, err error) {
				require.ErrorIs(t, err, apperrors.ErrForbidden)
			},
		},
		{
			name: "success from repository caches and returns ticket",
			setupMock: func(authorizer *mockservice.MockAuthorizer, ticketRepo *mockrepository.MockTicketRepository, ticketCache *mockservice.MockTicketCache, auditLogger *mockservice.MockAuditLogger) {
				ticket := entity.Ticket{ID: ticketID, TenantID: principal.TenantID, Subject: "repo", CreatedByUserID: principal.UserID}
				authorizer.EXPECT().Authorize(mock.Anything, principal, "tickets", "read").Return(nil)
				ticketCache.EXPECT().GetTicket(mock.Anything, ticketID).Return(entity.Ticket{}, false, nil)
				ticketRepo.EXPECT().GetByID(mock.Anything, ticketID).Return(ticket, nil)
				ticketCache.EXPECT().SetTicket(mock.Anything, ticket).Return(nil)
				auditLogger.EXPECT().Record(mock.Anything, mock.AnythingOfType("entity.AuditLog")).Return(nil)
			},
			assertFn: func(t *testing.T, ticket entity.Ticket, err error) {
				require.NoError(t, err)
				assert.Equal(t, "repo", ticket.Subject)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := mockservice.NewMockAuthorizer(t)
			ticketRepo := mockrepository.NewMockTicketRepository(t)
			ticketCache := mockservice.NewMockTicketCache(t)
			auditLogger := mockservice.NewMockAuditLogger(t)

			tt.setupMock(authorizer, ticketRepo, ticketCache, auditLogger)

			svc := query.NewGetTicketService(ticketRepo, authorizer, ticketCache, auditLogger)
			ticket, err := svc.Execute(context.Background(), queryInput)
			tt.assertFn(t, ticket, err)
		})
	}
}
