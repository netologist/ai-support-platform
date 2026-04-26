package command

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type UpdateTicketCommand struct {
	TicketID  uuid.UUID
	Principal entity.Principal
	Subject   *string
	Status    *string
}

type UpdateTicketService struct {
	ticketRepository repository.TicketRepository
	authorizer       service.Authorizer
	ticketCache      service.TicketCache
	auditLogger      service.AuditLogger
	publisher        service.MessagePublisher
	topic            string
}

func NewUpdateTicketService(
	ticketRepository repository.TicketRepository,
	authorizer service.Authorizer,
	ticketCache service.TicketCache,
	auditLogger service.AuditLogger,
	publisher service.MessagePublisher,
	topic string,
) UpdateTicketService {
	return UpdateTicketService{
		ticketRepository: ticketRepository,
		authorizer:       authorizer,
		ticketCache:      ticketCache,
		auditLogger:      auditLogger,
		publisher:        publisher,
		topic:            topic,
	}
}

func (svc UpdateTicketService) Execute(ctx context.Context, command UpdateTicketCommand) (entity.Ticket, error) {
	if err := svc.authorizer.Authorize(ctx, command.Principal, "tickets", "update"); err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			return entity.Ticket{}, apperrors.ErrForbidden
		}

		return entity.Ticket{}, err
	}

	if command.Subject == nil && command.Status == nil {
		return entity.Ticket{}, apperrors.ErrInvalidArgument
	}

	ticket, err := svc.ticketRepository.GetByID(ctx, command.TicketID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return entity.Ticket{}, apperrors.ErrNotFound
		}

		return entity.Ticket{}, err
	}

	if ticket.TenantID != command.Principal.TenantID {
		return entity.Ticket{}, apperrors.ErrForbidden
	}

	if command.Subject != nil {
		subject := strings.TrimSpace(*command.Subject)
		if subject == "" {
			return entity.Ticket{}, apperrors.ErrInvalidArgument
		}
		ticket.Subject = subject
	}

	if command.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*command.Status))
		if !entity.IsValidTicketStatus(status) {
			return entity.Ticket{}, apperrors.ErrInvalidArgument
		}
		ticket.Status = status
	}

	updatedTicket, err := svc.ticketRepository.Update(ctx, ticket)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return entity.Ticket{}, apperrors.ErrNotFound
		}

		return entity.Ticket{}, err
	}

	if svc.ticketCache != nil {
		_ = svc.ticketCache.SetTicket(ctx, updatedTicket)
	}

	svc.recordAudit(ctx, entity.AuditLog{
		EventType:  "ticket.updated",
		Action:     "update",
		Outcome:    "success",
		TenantID:   &command.Principal.TenantID,
		UserID:     &command.Principal.UserID,
		Resource:   "ticket",
		ResourceID: updatedTicket.ID.String(),
		Metadata: map[string]any{
			"status":  updatedTicket.Status,
			"subject": updatedTicket.Subject,
		},
	})

	if svc.publisher != nil && svc.topic != "" {
		_ = svc.publisher.PublishJSON(ctx, svc.topic, updatedTicket.ID.String(), map[string]any{
			"event_type": "ticket.updated",
			"ticket":     updatedTicket,
		})
	}

	return updatedTicket, nil
}

func (svc UpdateTicketService) recordAudit(ctx context.Context, entry entity.AuditLog) {
	if svc.auditLogger != nil {
		_ = svc.auditLogger.Record(ctx, entry)
	}
}
