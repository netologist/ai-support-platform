package repository

import (
	"context"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type AuthRepository interface {
	FindUserByEmail(ctx context.Context, email string) (entity.User, error)
}
