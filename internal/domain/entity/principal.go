package entity

import (
	"time"

	"github.com/google/uuid"
)

type Principal struct {
	UserID    uuid.UUID
	TenantID  uuid.UUID
	Email     string
	Role      string
	ExpiresAt time.Time
}
