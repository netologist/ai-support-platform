package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/netologist/ai-support-platform/internal/infra/messaging/worker"
)

type TicketEventHandler struct {
	idempotency worker.IdempotencyChecker
}

func NewTicketEventHandler(idempotency worker.IdempotencyChecker) *TicketEventHandler {
	return &TicketEventHandler{idempotency: idempotency}
}

func (h *TicketEventHandler) Handle(ctx context.Context, topic string, key []byte, value []byte) error {
	var envelope struct {
		EventType string `json:"EventType"`
	}
	if err := json.Unmarshal(value, &envelope); err != nil {
		return fmt.Errorf("unmarshal event envelope: %w", err)
	}

	idempotencyKey := fmt.Sprintf("%s:%s", topic, string(key))
	// CheckAndMark is atomic (Redis SET NX): eliminates TOCTOU race between
	// IsProcessed and MarkProcessed under concurrent consumers.
	isNew, err := h.idempotency.CheckAndMark(ctx, idempotencyKey)
	if err != nil {
		return fmt.Errorf("idempotency check-and-mark: %w", err)
	}
	if !isNew {
		slog.Info("event already processed, skipping", slog.String("key", idempotencyKey))
		return nil
	}

	switch envelope.EventType {
	case "ticket.created":
		// TODO: Implement ticket creation workflow (e.g., send notifications, update analytics)
		slog.Info("processing ticket.created", slog.String("key", string(key)))
	case "ticket.updated":
		// TODO: Implement ticket update workflow (e.g., send notifications, update related entities)
		slog.Info("processing ticket.updated", slog.String("key", string(key)))
	default:
		slog.Warn("unknown event type", slog.String("event_type", envelope.EventType))
	}

	// CheckAndMark already marked the event as processed atomically above.
	// No separate MarkProcessed call needed.

	return nil
}
