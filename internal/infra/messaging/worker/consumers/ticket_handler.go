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
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(value, &envelope); err != nil {
		return fmt.Errorf("unmarshal event envelope: %w", err)
	}

	idempotencyKey := fmt.Sprintf("%s:%s", topic, string(key))
	processed, err := h.idempotency.IsProcessed(ctx, idempotencyKey)
	if err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if processed {
		slog.Info("event already processed, skipping", slog.String("key", idempotencyKey))
		return nil
	}

	switch envelope.EventType {
	case "ticket.created":
		slog.Info("processing ticket.created", slog.String("key", string(key)))
	case "ticket.updated":
		slog.Info("processing ticket.updated", slog.String("key", string(key)))
	default:
		slog.Warn("unknown event type", slog.String("event_type", envelope.EventType))
	}

	if err := h.idempotency.MarkProcessed(ctx, idempotencyKey); err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	return nil
}
