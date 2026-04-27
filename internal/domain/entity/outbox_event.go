package entity

import (
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
	AttemptCount  int       // tracks failed publish attempts
	FailedAt      *time.Time // timestamp of last failure
}
