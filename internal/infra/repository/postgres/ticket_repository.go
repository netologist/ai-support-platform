package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	domainrepository "github.com/netologist/ai-support-platform/internal/domain/repository"
	generated "github.com/netologist/ai-support-platform/internal/infra/repository/sqlc"
)

type TicketRepository struct {
	queries *generated.Queries
}

func NewTicketRepository(queries *generated.Queries) TicketRepository {
	return TicketRepository{queries: queries}
}

func (repository TicketRepository) GetByID(ctx context.Context, ticketID uuid.UUID) (entity.Ticket, error) {
	ticket, err := repository.queries.GetTicketByID(ctx, toPGUUID(ticketID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Ticket{}, domainrepository.ErrNotFound
		}

		return entity.Ticket{}, err
	}

	return mapTicket(ticket.ID, ticket.TenantID, ticket.Subject, ticket.Status, ticket.CreatedByUserID, ticket.AssignedToUserID, ticket.CreatedAt)
}

func (repository TicketRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.Ticket, error) {
	rows, err := repository.queries.ListTicketsByTenant(ctx, toPGUUID(tenantID))
	if err != nil {
		return nil, err
	}

	tickets := make([]entity.Ticket, 0, len(rows))
	for _, row := range rows {
		ticket, err := mapTicket(row.ID, row.TenantID, row.Subject, row.Status, row.CreatedByUserID, row.AssignedToUserID, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

func (repository TicketRepository) Create(ctx context.Context, ticket entity.Ticket) (entity.Ticket, error) {
	createdAt := ticket.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	row, err := repository.queries.CreateTicket(ctx, generated.CreateTicketParams{
		ID:               toPGUUID(ticket.ID),
		TenantID:         toPGUUID(ticket.TenantID),
		Subject:          ticket.Subject,
		Status:           ticket.Status,
		CreatedByUserID:  toPGUUID(ticket.CreatedByUserID),
		AssignedToUserID: toNullablePGUUID(ticket.AssignedToUserID),
		CreatedAt:        toPGTimestamptz(createdAt),
	})
	if err != nil {
		return entity.Ticket{}, err
	}

	return mapTicket(row.ID, row.TenantID, row.Subject, row.Status, row.CreatedByUserID, row.AssignedToUserID, row.CreatedAt)
}

func (repository TicketRepository) Update(ctx context.Context, ticket entity.Ticket) (entity.Ticket, error) {
	row, err := repository.queries.UpdateTicket(ctx, generated.UpdateTicketParams{
		ID:               toPGUUID(ticket.ID),
		Subject:          ticket.Subject,
		Status:           ticket.Status,
		AssignedToUserID: toNullablePGUUID(ticket.AssignedToUserID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Ticket{}, domainrepository.ErrNotFound
		}

		return entity.Ticket{}, err
	}

	return mapTicket(row.ID, row.TenantID, row.Subject, row.Status, row.CreatedByUserID, row.AssignedToUserID, row.CreatedAt)
}

func (repository TicketRepository) Delete(ctx context.Context, ticketID uuid.UUID) error {
	return repository.queries.DeleteTicket(ctx, toPGUUID(ticketID))
}

func mapTicket(id pgtype.UUID, tenantID pgtype.UUID, subject string, status string, createdByUserID pgtype.UUID, assignedToUserID pgtype.UUID, createdAt pgtype.Timestamptz) (entity.Ticket, error) {
	domainTicketID, err := toDomainUUID(id)
	if err != nil {
		return entity.Ticket{}, err
	}

	domainTenantID, err := toDomainUUID(tenantID)
	if err != nil {
		return entity.Ticket{}, err
	}

	domainCreatedByUserID, err := toDomainUUID(createdByUserID)
	if err != nil {
		return entity.Ticket{}, err
	}

	var domainAssignedToUserID *uuid.UUID
	if assignedToUserID.Valid {
		v, err := toDomainUUID(assignedToUserID)
		if err != nil {
			return entity.Ticket{}, err
		}
		domainAssignedToUserID = &v
	}

	domainCreatedAt, err := toTime(createdAt)
	if err != nil {
		return entity.Ticket{}, err
	}

	return entity.Ticket{
		ID:               domainTicketID,
		TenantID:         domainTenantID,
		Subject:          subject,
		Status:           status,
		CreatedByUserID:  domainCreatedByUserID,
		AssignedToUserID: domainAssignedToUserID,
		CreatedAt:        domainCreatedAt,
	}, nil
}
