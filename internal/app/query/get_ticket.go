package query

import (
	"context"
	"errors"

	"github.com/google/uuid"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type GetTicketQuery struct {
	TicketID   uuid.UUID
	Principal  entity.Principal
	Resource   string
	ActionName string
}

type GetTicketService struct {
	ticketRepository repository.TicketRepository
	authorizer       service.Authorizer
	ticketCache      service.TicketCache
	auditLogger      service.AuditLogger
}

func NewGetTicketService(ticketRepository repository.TicketRepository, authorizer service.Authorizer, ticketCache service.TicketCache, auditLogger service.AuditLogger) GetTicketService {
	return GetTicketService{ticketRepository: ticketRepository, authorizer: authorizer, ticketCache: ticketCache, auditLogger: auditLogger}
}

func (svc GetTicketService) Execute(ctx context.Context, query GetTicketQuery) (entity.Ticket, error) {
	if err := svc.authorizer.Authorize(ctx, query.Principal, query.Resource, query.ActionName); err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			svc.recordAudit(ctx, entity.AuditLog{EventType: "ticket.read", Action: "read", Outcome: "forbidden", TenantID: &query.Principal.TenantID, UserID: &query.Principal.UserID, Resource: "ticket", ResourceID: query.TicketID.String()})
			return entity.Ticket{}, apperrors.ErrForbidden
		}

		return entity.Ticket{}, err
	}

	if svc.ticketCache != nil {
		cachedTicket, found, err := svc.ticketCache.GetTicket(ctx, query.TicketID)
		if err == nil && found {
			if cachedTicket.TenantID != query.Principal.TenantID {
				return entity.Ticket{}, apperrors.ErrForbidden
			}

			svc.recordAudit(ctx, entity.AuditLog{EventType: "ticket.read", Action: "read", Outcome: "success", TenantID: &query.Principal.TenantID, UserID: &query.Principal.UserID, Resource: "ticket", ResourceID: query.TicketID.String(), Metadata: map[string]any{"cache_hit": true}})
			return cachedTicket, nil
		}
	}

	ticket, err := svc.ticketRepository.GetByID(ctx, query.TicketID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			svc.recordAudit(ctx, entity.AuditLog{EventType: "ticket.read", Action: "read", Outcome: "not_found", TenantID: &query.Principal.TenantID, UserID: &query.Principal.UserID, Resource: "ticket", ResourceID: query.TicketID.String()})
			return entity.Ticket{}, apperrors.ErrNotFound
		}

		return entity.Ticket{}, err
	}

	if ticket.TenantID != query.Principal.TenantID {
		svc.recordAudit(ctx, entity.AuditLog{EventType: "ticket.read", Action: "read", Outcome: "forbidden", TenantID: &query.Principal.TenantID, UserID: &query.Principal.UserID, Resource: "ticket", ResourceID: query.TicketID.String()})
		return entity.Ticket{}, apperrors.ErrForbidden
	}

	if svc.ticketCache != nil {
		_ = svc.ticketCache.SetTicket(ctx, ticket)
	}

	svc.recordAudit(ctx, entity.AuditLog{EventType: "ticket.read", Action: "read", Outcome: "success", TenantID: &query.Principal.TenantID, UserID: &query.Principal.UserID, Resource: "ticket", ResourceID: ticket.ID.String(), Metadata: map[string]any{"cache_hit": false}})

	return ticket, nil
}

func (svc GetTicketService) recordAudit(ctx context.Context, entry entity.AuditLog) {
	if svc.auditLogger != nil {
		_ = svc.auditLogger.Record(ctx, entry)
	}
}
