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
	"github.com/netologist/ai-support-platform/internal/domain/service"
	mockrepo "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mocksvc "github.com/netologist/ai-support-platform/internal/mocks/service"
)

func TestCreateTicketService_Execute(t *testing.T) {
	fixedTenantID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	fixedUserID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

	fixedPrincipal := entity.Principal{
		UserID:   fixedUserID,
		TenantID: fixedTenantID,
		Email:    "agent@example.com",
		Role:     "agent",
	}

	newTicketCmd := command.CreateTicketCommand{
		Principal: fixedPrincipal,
		Subject:   "My ticket",
	}

	tests := []struct {
		name        string
		setupMocks  func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher)
		useNilDeps  bool
		wantSubject string
		wantErr     error
	}{
		{
			name: "creates ticket and returns it",
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher) {
				auth.EXPECT().Authorize(mock.Anything, fixedPrincipal, "tickets", "create").Return(nil)
				repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t entity.Ticket) bool {
					return t.Subject == "My ticket" &&
						t.TenantID == fixedTenantID &&
						t.CreatedByUserID == fixedUserID &&
						t.Status == entity.TicketStatusOpen
				})).Return(entity.Ticket{
					ID:              uuid.New(),
					TenantID:        fixedTenantID,
					Subject:         "My ticket",
					Status:          entity.TicketStatusOpen,
					CreatedByUserID: fixedUserID,
				}, nil)
				cache.EXPECT().SetTicket(mock.Anything, mock.Anything).Return(nil)
				auditor.EXPECT().Record(mock.Anything, mock.MatchedBy(func(log entity.AuditLog) bool {
					return log.EventType == "ticket.created" && log.Outcome == "success"
				})).Return(nil)
				pub.EXPECT().PublishJSON(mock.Anything, "tickets", mock.Anything, mock.Anything).Return(nil)
			},
			wantSubject: "My ticket",
		},
		{
			name: "authorization denied returns ErrForbidden",
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher) {
				auth.EXPECT().Authorize(mock.Anything, fixedPrincipal, "tickets", "create").Return(service.ErrPermissionDenied)
			},
			wantErr: apperrors.ErrForbidden,
		},
		{
			name: "authorization error propagates",
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher) {
				auth.EXPECT().Authorize(mock.Anything, fixedPrincipal, "tickets", "create").Return(errors.New("authz service unavailable"))
			},
			wantErr: errors.New("authz service unavailable"),
		},
		{
			name: "repository error propagates",
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher) {
				auth.EXPECT().Authorize(mock.Anything, fixedPrincipal, "tickets", "create").Return(nil)
				repo.EXPECT().Create(mock.Anything, mock.Anything).Return(entity.Ticket{}, errors.New("db write failed"))
			},
			wantErr: errors.New("db write failed"),
		},
		{
			name: "nil cache and publisher are safe to omit",
			setupMocks: func(repo *mockrepo.MockTicketRepository, auth *mocksvc.MockAuthorizer, cache *mocksvc.MockTicketCache, auditor *mocksvc.MockAuditLogger, pub *mocksvc.MockMessagePublisher) {
				auth.EXPECT().Authorize(mock.Anything, fixedPrincipal, "tickets", "create").Return(nil)
				repo.EXPECT().Create(mock.Anything, mock.Anything).Return(entity.Ticket{
					ID:      uuid.New(),
					Subject: "My ticket",
					Status:  entity.TicketStatusOpen,
				}, nil)
				auditor.EXPECT().Record(mock.Anything, mock.Anything).Return(nil)
			},
			useNilDeps:  true,
			wantSubject: "My ticket",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mockrepo.NewMockTicketRepository(t)
			auth := mocksvc.NewMockAuthorizer(t)
			auditor := mocksvc.NewMockAuditLogger(t)

			var cache *mocksvc.MockTicketCache
			var pub *mocksvc.MockMessagePublisher
			if !tc.useNilDeps {
				cache = mocksvc.NewMockTicketCache(t)
				pub = mocksvc.NewMockMessagePublisher(t)
			}

			tc.setupMocks(repo, auth, cache, auditor, pub)

			// Avoid typed-nil interface trap: pass untyped nil when deps are omitted.
			var svc command.CreateTicketService
			if tc.useNilDeps {
				svc = command.NewCreateTicketService(repo, auth, nil, auditor, nil, "")
			} else {
				svc = command.NewCreateTicketService(repo, auth, cache, auditor, pub, "tickets")
			}

			ticket, err := svc.Execute(context.Background(), newTicketCmd)

			if tc.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tc.wantErr, apperrors.ErrForbidden) {
					assert.ErrorIs(t, err, apperrors.ErrForbidden)
				} else {
					assert.EqualError(t, err, tc.wantErr.Error())
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantSubject, ticket.Subject)
		})
	}
}
