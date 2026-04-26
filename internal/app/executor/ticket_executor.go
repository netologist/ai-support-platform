package executor

import (
	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

// Convenience aliases for concrete ticket command executors.
type CreateTicketExecutor Executor[command.CreateTicketCommand, entity.Ticket]
type UpdateTicketExecutor Executor[command.UpdateTicketCommand, entity.Ticket]
type GetTicketExecutor Executor[query.GetTicketQuery, entity.Ticket]
type ListTicketsExecutor Executor[query.ListTicketsQuery, []entity.Ticket]
