package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type Relay struct {
	outbox    repository.OutboxRepository
	publisher service.MessagePublisher
	topic     string
	batchSize int
	interval  time.Duration
}

func NewRelay(
	outbox repository.OutboxRepository,
	publisher service.MessagePublisher,
	topic string,
	batchSize int,
	interval time.Duration,
) *Relay {
	return &Relay{
		outbox:    outbox,
		publisher: publisher,
		topic:     topic,
		batchSize: batchSize,
		interval:  interval,
	}
}

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.flush(ctx); err != nil {
				slog.Error("outbox relay flush failed", slog.Any("error", err))
			}
		}
	}
}

func (r *Relay) flush(ctx context.Context) error {
	events, err := r.outbox.GetUnsentEvents(ctx, r.batchSize)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := r.publisher.PublishJSON(ctx, r.topic, event.AggregateID.String(), event); err != nil {
			slog.Error("outbox relay publish failed",
				slog.String("event_id", event.ID.String()),
				slog.Any("error", err),
			)
			continue
		}

		if err := r.outbox.MarkEventSent(ctx, event.ID.String()); err != nil {
			slog.Error("outbox relay mark sent failed",
				slog.String("event_id", event.ID.String()),
				slog.Any("error", err),
			)
		}
	}

	return nil
}

func (r *Relay) FlushOnce(ctx context.Context) error {
	return r.flush(ctx)
}
