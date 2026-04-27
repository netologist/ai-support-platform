package command

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

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
	outbox           repository.OutboxRepository
}

func NewUpdateTicketService(
	ticketRepository repository.TicketRepository,
	authorizer service.Authorizer,
	ticketCache service.TicketCache,
	auditLogger service.AuditLogger,
	outbox repository.OutboxRepository,
) UpdateTicketService {
	return UpdateTicketService{
		ticketRepository: ticketRepository,
		authorizer:       authorizer,
		ticketCache:      ticketCache,
		auditLogger:      auditLogger,
		outbox:           outbox,
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

	// Write outbox event for ticket.updated
	if svc.outbox != nil {
		payload, _ := json.Marshal(map[string]any{
			"event_type": "ticket.updated",
			"ticket":     updatedTicket,
		})
		if err := svc.outbox.InsertEvent(ctx, &entity.OutboxEvent{
			ID:            uuid.New(),
			AggregateType: "ticket",
			AggregateID:   updatedTicket.ID,
			EventType:     "ticket.updated",
			Payload:       payload,
			CreatedAt:     time.Now().UTC(),
		}); err != nil {
			slog.Error("outbox insert failed after ticket update",
				slog.String("ticket_id", updatedTicket.ID.String()),
				slog.Any("error", err),
			)
		}
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

	return updatedTicket, nil
}

func (svc UpdateTicketService) recordAudit(ctx context.Context, entry entity.AuditLog) {
	if svc.auditLogger != nil {
		_ = svc.auditLogger.Record(ctx, entry)
	}
}
