package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	domainrepository "github.com/netologist/ai-support-platform/internal/domain/repository"
	generated "github.com/netologist/ai-support-platform/internal/infra/repository/sqlc"
)

type AuthRepository struct {
	queries *generated.Queries
}

func NewAuthRepository(queries *generated.Queries) AuthRepository {
	return AuthRepository{queries: queries}
}

func (repository AuthRepository) FindUserByEmail(ctx context.Context, email string) (entity.User, error) {
	user, err := repository.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.User{}, domainrepository.ErrNotFound
		}

		return entity.User{}, err
	}

	userID, err := toDomainUUID(user.ID)
	if err != nil {
		return entity.User{}, err
	}

	createdAt, err := toTime(user.CreatedAt)
	if err != nil {
		return entity.User{}, err
	}

	return entity.User{
		ID:           userID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    createdAt,
	}, nil
}

