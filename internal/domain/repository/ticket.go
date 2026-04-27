package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type TicketRepository interface {
	GetByID(ctx context.Context, ticketID uuid.UUID) (entity.Ticket, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.Ticket, error)
	Create(ctx context.Context, ticket entity.Ticket) (entity.Ticket, error)
	Update(ctx context.Context, ticket entity.Ticket) (entity.Ticket, error)
	Delete(ctx context.Context, ticketID uuid.UUID) error
}
