package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
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

func (r AuthRepository) FindUserByEmail(ctx context.Context, email string) (entity.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
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

func (r AuthRepository) FindMembership(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (entity.Membership, error) {
	membership, err := r.queries.GetMembershipByUserAndTenant(ctx, generated.GetMembershipByUserAndTenantParams{
		UserID:   toPGUUID(userID),
		TenantID: toPGUUID(tenantID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Membership{}, domainrepository.ErrNotFound
		}

		return entity.Membership{}, err
	}

	domainTenantID, err := toDomainUUID(membership.TenantID)
	if err != nil {
		return entity.Membership{}, err
	}

	domainUserID, err := toDomainUUID(membership.UserID)
	if err != nil {
		return entity.Membership{}, err
	}

	createdAt, err := toTime(membership.CreatedAt)
	if err != nil {
		return entity.Membership{}, err
	}

	return entity.Membership{
		TenantID:  domainTenantID,
		UserID:    domainUserID,
		Role:      membership.Role,
		CreatedAt: createdAt,
	}, nil
}
