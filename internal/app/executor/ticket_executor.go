package executor

import (
	"context"

	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

// TicketExecutor defines a generic command executor for ticket operations.
type TicketExecutor[T any, R any] interface {
	Execute(ctx context.Context, cmd T) (R, error)
}

// Convenience aliases for concrete ticket command executors.
type CreateTicketExecutor = TicketExecutor[command.CreateTicketCommand, entity.Ticket]
type UpdateTicketExecutor = TicketExecutor[command.UpdateTicketCommand, entity.Ticket]
type GetTicketExecutor = TicketExecutor[query.GetTicketQuery, entity.Ticket]
type ListTicketsExecutor = TicketExecutor[query.ListTicketsQuery, []entity.Ticket]
