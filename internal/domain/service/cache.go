package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type TicketCache interface {
	GetTicket(ctx context.Context, ticketID uuid.UUID) (entity.Ticket, bool, error)
	SetTicket(ctx context.Context, ticket entity.Ticket) error
	DeleteTicket(ctx context.Context, ticketID uuid.UUID) error
}
