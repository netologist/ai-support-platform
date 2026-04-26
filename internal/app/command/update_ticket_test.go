package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/app/command"
	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	mockrepo "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mocksvc "github.com/netologist/ai-support-platform/internal/mocks/service"
)

func ptr[T any](v T) *T { return &v }

func TestUpdateTicketService_Execute(t *testing.T) {
	fixedTenantID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	fixedUserID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	fixedTicketID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	otherTenantID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	principal := entity.Principal{
		UserID:   fixedUserID,
		TenantID: fixedTenantID,
		Email:    "agent@example.com",
		Role:     "agent",
	}

	existingTicket := entity.Ticket{
		ID:              fixedTicketID,
		TenantID:        fixedTenantID,
		Subject:         "Original subject",
		Status:          entity.TicketStatusOpen,
		CreatedByUserID: fixedUserID,
	}

	tests := []struct {
		name       string
		cmd        command.UpdateTicketCommand
		setupMocks func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository)
		useNilDeps bool
		wantErr    error
		check      func(t *testing.T, ticket entity.Ticket)
	}{
		{
			name: "updates subject successfully",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Subject:   ptr("New subject"),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
				repo.EXPECT().GetByID(mock.Anything, fixedTicketID).Return(existingTicket, nil)
				repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(t entity.Ticket) bool {
					return t.Subject == "New subject"
				})).Return(entity.Ticket{
					ID:       fixedTicketID,
					TenantID: fixedTenantID,
					Subject:  "New subject",
					Status:   entity.TicketStatusOpen,
				}, nil)
				outbox.EXPECT().InsertEvent(mock.Anything, mock.Anything).Return(nil)
				cache.EXPECT().SetTicket(mock.Anything, mock.Anything).Return(nil)
				auditor.EXPECT().Record(mock.Anything, mock.MatchedBy(func(l entity.AuditLog) bool {
					return l.EventType == "ticket.updated" && l.Outcome == "success"
				})).Return(nil)
				pub.EXPECT().PublishJSON(mock.Anything, "tickets", fixedTicketID.String(), mock.Anything).Return(nil)
			},
			check: func(t *testing.T, ticket entity.Ticket) {
				assert.Equal(t, "New subject", ticket.Subject)
			},
		},
		{
			name: "updates status successfully",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Status:    ptr(entity.TicketStatusClosed),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
				repo.EXPECT().GetByID(mock.Anything, fixedTicketID).Return(existingTicket, nil)
				repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(t entity.Ticket) bool {
					return t.Status == entity.TicketStatusClosed
				})).Return(entity.Ticket{
					ID:       fixedTicketID,
					TenantID: fixedTenantID,
					Subject:  existingTicket.Subject,
					Status:   entity.TicketStatusClosed,
				}, nil)
				outbox.EXPECT().InsertEvent(mock.Anything, mock.Anything).Return(nil)
				cache.EXPECT().SetTicket(mock.Anything, mock.Anything).Return(nil)
				auditor.EXPECT().Record(mock.Anything, mock.Anything).Return(nil)
				pub.EXPECT().PublishJSON(mock.Anything, "tickets", fixedTicketID.String(), mock.Anything).Return(nil)
			},
			check: func(t *testing.T, ticket entity.Ticket) {
				assert.Equal(t, entity.TicketStatusClosed, ticket.Status)
			},
		},
		{
			name: "nil cache and publisher are skipped without panic",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Subject:   ptr("Updated"),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
				repo.EXPECT().GetByID(mock.Anything, fixedTicketID).Return(existingTicket, nil)
				repo.EXPECT().Update(mock.Anything, mock.Anything).Return(entity.Ticket{
					ID:       fixedTicketID,
					TenantID: fixedTenantID,
					Subject:  "Updated",
					Status:   entity.TicketStatusOpen,
				}, nil)
				auditor.EXPECT().Record(mock.Anything, mock.Anything).Return(nil)
			},
			useNilDeps: true,
		},
		{
			name: "permission denied returns ErrForbidden",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Subject:   ptr("x"),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(service.ErrPermissionDenied)
			},
			wantErr: apperrors.ErrForbidden,
		},
		{
			name: "authorizer error propagates",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Subject:   ptr("x"),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(errors.New("authz error"))
			},
			wantErr: errors.New("authz error"),
		},
		{
			name: "no fields provided returns ErrInvalidArgument",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
			},
			wantErr: apperrors.ErrInvalidArgument,
		},
		{
			name: "ticket not found returns ErrNotFound",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Subject:   ptr("x"),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
				repo.EXPECT().GetByID(mock.Anything, fixedTicketID).Return(entity.Ticket{}, repository.ErrNotFound)
			},
			wantErr: apperrors.ErrNotFound,
		},
		{
			name: "ticket belongs to different tenant returns ErrForbidden",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Subject:   ptr("x"),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
				repo.EXPECT().GetByID(mock.Anything, fixedTicketID).Return(entity.Ticket{
					ID:       fixedTicketID,
					TenantID: otherTenantID,
				}, nil)
			},
			wantErr: apperrors.ErrForbidden,
		},
		{
			name: "blank subject returns ErrInvalidArgument",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Subject:   ptr("   "),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
				repo.EXPECT().GetByID(mock.Anything, fixedTicketID).Return(existingTicket, nil)
			},
			wantErr: apperrors.ErrInvalidArgument,
		},
		{
			name: "invalid status returns ErrInvalidArgument",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Status:    ptr("pending"),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
				repo.EXPECT().GetByID(mock.Anything, fixedTicketID).Return(existingTicket, nil)
			},
			wantErr: apperrors.ErrInvalidArgument,
		},
		{
			name: "repository Update error propagates",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Subject:   ptr("x"),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
				repo.EXPECT().GetByID(mock.Anything, fixedTicketID).Return(existingTicket, nil)
				repo.EXPECT().Update(mock.Anything, mock.Anything).Return(entity.Ticket{}, errors.New("db error"))
			},
			wantErr: errors.New("db error"),
		},
		{
			name: "repository Update not found returns ErrNotFound",
			cmd: command.UpdateTicketCommand{
				TicketID:  fixedTicketID,
				Principal: principal,
				Subject:   ptr("x"),
			},
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher, outbox *mockrepo.MockOutboxRepository) {
				auth.EXPECT().Authorize(mock.Anything, principal, "tickets", "update").Return(nil)
				repo.EXPECT().GetByID(mock.Anything, fixedTicketID).Return(existingTicket, nil)
				repo.EXPECT().Update(mock.Anything, mock.Anything).Return(entity.Ticket{}, repository.ErrNotFound)
			},
			wantErr: apperrors.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mockrepo.NewMockTicketRepository(t)
			auth := mocksvc.NewMockAuthorizer(t)
			auditor := mocksvc.NewMockAuditLogger(t)
			outbox := mockrepo.NewMockOutboxRepository(t)

			var cache *mocksvc.MockTicketCache
			var pub *mocksvc.MockMessagePublisher
			if !tc.useNilDeps {
				cache = mocksvc.NewMockTicketCache(t)
				pub = mocksvc.NewMockMessagePublisher(t)
			}

			tc.setupMocks(repo, auth, cache, auditor, pub, outbox)

			// Avoid typed-nil interface trap: pass untyped nil when deps are omitted.
			var svc command.UpdateTicketService
			if tc.useNilDeps {
				svc = command.NewUpdateTicketService(repo, auth, nil, auditor, nil, "", nil)
			} else {
				svc = command.NewUpdateTicketService(repo, auth, cache, auditor, pub, "tickets", outbox)
			}

			ticket, err := svc.Execute(context.Background(), tc.cmd)

			if tc.wantErr != nil {
				require.Error(t, err)
				switch {
				case errors.Is(tc.wantErr, apperrors.ErrForbidden):
					assert.ErrorIs(t, err, apperrors.ErrForbidden)
				case errors.Is(tc.wantErr, apperrors.ErrNotFound):
					assert.ErrorIs(t, err, apperrors.ErrNotFound)
				case errors.Is(tc.wantErr, apperrors.ErrInvalidArgument):
					assert.ErrorIs(t, err, apperrors.ErrInvalidArgument)
				default:
					assert.EqualError(t, err, tc.wantErr.Error())
				}
				return
			}

			require.NoError(t, err)
			if tc.check != nil {
				tc.check(t, ticket)
			}
		})
	}
}
