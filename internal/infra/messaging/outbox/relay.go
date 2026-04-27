package outbox

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

const maxRetries = 5

type Relay struct {
	outbox           repository.OutboxRepository
	publisher        service.MessagePublisher
	topic            string
	batchSize        int
	interval         time.Duration
	failureTracker   map[string]int // event ID -> failure count
	failureTrackerMu sync.Mutex
}

func NewRelay(
	outbox repository.OutboxRepository,
	publisher service.MessagePublisher,
	topic string,
	batchSize int,
	interval time.Duration,
) *Relay {
	return &Relay{
		outbox:         outbox,
		publisher:      publisher,
		topic:          topic,
		batchSize:      batchSize,
		interval:       interval,
		failureTracker: make(map[string]int),
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
		eventIDStr := event.ID.String()

		r.failureTrackerMu.Lock()
		attempts := r.failureTracker[eventIDStr]
		r.failureTrackerMu.Unlock()

		if attempts >= maxRetries {
			// Dead letter: mark event as sent but log warning
			slog.Warn("outbox relay max retries exceeded, discarding event",
				slog.String("event_id", eventIDStr),
				slog.Int("attempts", attempts),
			)
			if err := r.outbox.MarkEventSent(ctx, event.ID); err != nil {
				slog.Error("outbox relay failed to mark dead letter event as sent",
					slog.String("event_id", eventIDStr),
					slog.Any("error", err),
				)
			}
			r.failureTrackerMu.Lock()
			delete(r.failureTracker, eventIDStr)
			r.failureTrackerMu.Unlock()
			continue
		}

		if err := r.publisher.PublishJSON(ctx, r.topic, event.AggregateID.String(), event); err != nil {
			r.failureTrackerMu.Lock()
			r.failureTracker[eventIDStr]++
			r.failureTrackerMu.Unlock()

			slog.Error("outbox relay publish failed",
				slog.String("event_id", eventIDStr),
				slog.Int("attempt", attempts+1),
				slog.Int("max_retries", maxRetries),
				slog.Any("error", err),
			)
			continue
		}

		if err := r.outbox.MarkEventSent(ctx, event.ID); err != nil {
			slog.Error("outbox relay mark sent failed",
				slog.String("event_id", eventIDStr),
				slog.Any("error", err),
			)
		} else {
			// Clear failure tracker on success
			r.failureTrackerMu.Lock()
			delete(r.failureTracker, eventIDStr)
			r.failureTrackerMu.Unlock()
		}
	}

	return nil
}

func (r *Relay) FlushOnce(ctx context.Context) error {
	return r.flush(ctx)
}
