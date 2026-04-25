package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID
	OccurredAt time.Time
	EventType  string
	Action     string
	Outcome    string
	TenantID   *uuid.UUID
	UserID     *uuid.UUID
	Resource   string
	ResourceID string
	Metadata   map[string]any
}
