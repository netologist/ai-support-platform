package entity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
type Membership struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	Role      string
	CreatedAt time.Time
}
