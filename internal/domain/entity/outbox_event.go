package entity

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   uuid.UUID
	EventType     string
	Payload       []byte // JSON
	CreatedAt     time.Time
	SentAt        *time.Time
}

// NewOutboxEvent creates an OutboxEvent and validates that payload is well-formed JSON.
func NewOutboxEvent(aggregateType string, aggregateID uuid.UUID, eventType string, payload any) (*OutboxEvent, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("outbox event payload must be JSON-serialisable: %w", err)
	}
	return &OutboxEvent{
		ID:            uuid.New(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       raw,
		CreatedAt:     time.Now().UTC(),
	}, nil
}
