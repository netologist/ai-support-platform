package repository

import (
	"context"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type OutboxRepository interface {
	InsertEvent(ctx context.Context, event *entity.OutboxEvent) error
	GetUnsentEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error)
	MarkEventSent(ctx context.Context, id string) error
}
