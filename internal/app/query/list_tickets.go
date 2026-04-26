package query

import (
	"context"
	"errors"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type ListTicketsQuery struct {
	Principal entity.Principal
}

type ListTicketsService struct {
	ticketRepository repository.TicketRepository
	authorizer       service.Authorizer
	auditLogger      service.AuditLogger
}

func NewListTicketsService(ticketRepository repository.TicketRepository, authorizer service.Authorizer, auditLogger service.AuditLogger) ListTicketsService {
	return ListTicketsService{ticketRepository: ticketRepository, authorizer: authorizer, auditLogger: auditLogger}
}

func (svc ListTicketsService) Execute(ctx context.Context, query ListTicketsQuery) ([]entity.Ticket, error) {
	if err := svc.authorizer.Authorize(ctx, query.Principal, "tickets", "read"); err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			return nil, apperrors.ErrForbidden
		}

		return nil, err
	}

	tickets, err := svc.ticketRepository.ListByTenant(ctx, query.Principal.TenantID)
	if err != nil {
		return nil, err
	}

	if svc.auditLogger != nil {
		_ = svc.auditLogger.Record(ctx, entity.AuditLog{
			EventType: "ticket.list",
			Action:    "list",
			Outcome:   "success",
			TenantID:  &query.Principal.TenantID,
			UserID:    &query.Principal.UserID,
			Resource:  "ticket",
			Metadata:  map[string]any{"count": len(tickets)},
		})
	}

	return tickets, nil
}
