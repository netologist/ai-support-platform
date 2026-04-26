package service

import (
	"context"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type OutboxWriter interface {
	WriteEvent(ctx context.Context, event *entity.OutboxEvent) error
}
