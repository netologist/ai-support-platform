package command

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type CreateTicketCommand struct {
	Principal entity.Principal
	Subject   string
}

type CreateTicketService struct {
	ticketRepository repository.TicketRepository
	authorizer       service.Authorizer
	ticketCache      service.TicketCache
	auditLogger      service.AuditLogger
	outbox           repository.OutboxRepository
}

func NewCreateTicketService(
	ticketRepository repository.TicketRepository,
	authorizer service.Authorizer,
	ticketCache service.TicketCache,
	auditLogger service.AuditLogger,
	outbox repository.OutboxRepository,
) CreateTicketService {
	return CreateTicketService{
		ticketRepository: ticketRepository,
		authorizer:       authorizer,
		ticketCache:      ticketCache,
		auditLogger:      auditLogger,
		outbox:           outbox,
	}
}

func (svc CreateTicketService) Execute(ctx context.Context, command CreateTicketCommand) (entity.Ticket, error) {
	if err := svc.authorizer.Authorize(ctx, command.Principal, "tickets", "create"); err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			return entity.Ticket{}, apperrors.ErrForbidden
		}

		return entity.Ticket{}, err
	}

	now := time.Now().UTC()
	ticket := entity.Ticket{
		ID:              uuid.New(),
		TenantID:        command.Principal.TenantID,
		Subject:         command.Subject,
		Status:          entity.TicketStatusOpen,
		CreatedByUserID: command.Principal.UserID,
		CreatedAt:       now,
	}

	createdTicket, err := svc.ticketRepository.Create(ctx, ticket)
	if err != nil {
		return entity.Ticket{}, err
	}

	// Write outbox event for ticket.created
	if svc.outbox != nil {
		payload, _ := json.Marshal(map[string]any{
			"event_type": "ticket.created",
			"ticket":     createdTicket,
		})
		if err := svc.outbox.InsertEvent(ctx, &entity.OutboxEvent{
			ID:            uuid.New(),
			AggregateType: "ticket",
			AggregateID:   createdTicket.ID,
			EventType:     "ticket.created",
			Payload:       payload,
			CreatedAt:     now,
		}); err != nil {
			slog.Error("outbox insert failed after ticket create",
				slog.String("ticket_id", createdTicket.ID.String()),
				slog.Any("error", err),
			)
		}
	}

	if svc.ticketCache != nil {
		_ = svc.ticketCache.SetTicket(ctx, createdTicket)
	}

	svc.recordAudit(ctx, entity.AuditLog{
		EventType:  "ticket.created",
		Action:     "create",
		Outcome:    "success",
		TenantID:   &command.Principal.TenantID,
		UserID:     &command.Principal.UserID,
		Resource:   "ticket",
		ResourceID: createdTicket.ID.String(),
		Metadata:   map[string]any{"status": createdTicket.Status},
	})

	return createdTicket, nil
}

func (svc CreateTicketService) recordAudit(ctx context.Context, entry entity.AuditLog) {
	if svc.auditLogger != nil {
		_ = svc.auditLogger.Record(ctx, entry)
	}
}
