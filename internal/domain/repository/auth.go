package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type AuthRepository interface {
	FindUserByEmail(ctx context.Context, email string) (entity.User, error)
	FindMembership(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (entity.Membership, error)
}
