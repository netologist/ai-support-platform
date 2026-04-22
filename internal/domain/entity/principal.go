package entity

import (
	"time"

	"github.com/google/uuid"
)

type Principal struct {
	UserID    uuid.UUID
	Email     string
	ExpiresAt time.Time
}
