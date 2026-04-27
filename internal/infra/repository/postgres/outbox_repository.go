package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	generated "github.com/netologist/ai-support-platform/internal/infra/repository/sqlc"
)

type OutboxRepository struct {
	queries *generated.Queries
}

func NewOutboxRepository(queries *generated.Queries) repository.OutboxRepository {
	return &OutboxRepository{queries: queries}
}

func (r *OutboxRepository) InsertEvent(ctx context.Context, event *entity.OutboxEvent) error {
	return r.queries.InsertOutboxEvent(ctx, generated.InsertOutboxEventParams{
		ID:            toPGUUID(event.ID),
		AggregateType: event.AggregateType,
		AggregateID:   toPGUUID(event.AggregateID),
		EventType:     event.EventType,
		Payload:       event.Payload,
	})
}

func (r *OutboxRepository) GetUnsentEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	rows, err := r.queries.GetUnsentOutboxEvents(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	events := make([]*entity.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		id, err := toDomainUUID(row.ID)
		if err != nil {
			return nil, err
		}
		aggID, err := toDomainUUID(row.AggregateID)
		if err != nil {
			return nil, err
		}
		t, err := toTime(row.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, &entity.OutboxEvent{
			ID:            id,
			AggregateType: row.AggregateType,
			AggregateID:   aggID,
			EventType:     row.EventType,
			Payload:       row.Payload,
			CreatedAt:     t,
		})
	}

	return events, nil
}

func (r *OutboxRepository) MarkEventSent(ctx context.Context, id uuid.UUID) error {
	return r.queries.MarkOutboxEventSent(ctx, toPGUUID(id))
}
