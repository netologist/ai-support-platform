package entity

import (
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	Subject          string
	Status           string
	CreatedByUserID  uuid.UUID
	AssignedToUserID *uuid.UUID
	CreatedAt        time.Time
}

const TicketStatusOpen = "open"

const TicketStatusClosed = "closed"

func IsValidTicketStatus(status string) bool {
	switch status {
	case TicketStatusOpen, TicketStatusClosed:
		return true
	default:
		return false
	}
}
