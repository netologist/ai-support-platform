package command

import (
	"context"
	"errors"

	"github.com/google/uuid"
	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type LoginExecutor interface {
	Execute(ctx context.Context, cmd LoginCommand) (LoginResult, error)
}

type LoginCommand struct {
	Email    string
	Password string
	TenantID uuid.UUID
}

type LoginResult struct {
	AccessToken string
	Principal   entity.Principal
}

type LoginService struct {
	authRepository   repository.AuthRepository
	passwordVerifier service.PasswordVerifier
	tokenIssuer      service.TokenIssuer
	auditLogger      service.AuditLogger
}

func NewLoginService(
	authRepository repository.AuthRepository,
	passwordVerifier service.PasswordVerifier,
	tokenIssuer service.TokenIssuer,
	auditLogger service.AuditLogger,
) *LoginService {
	return &LoginService{
		authRepository:   authRepository,
		passwordVerifier: passwordVerifier,
		tokenIssuer:      tokenIssuer,
		auditLogger:      auditLogger,
	}
}

func (s *LoginService) Execute(ctx context.Context, command LoginCommand) (LoginResult, error) {
	user, err := s.authRepository.FindUserByEmail(ctx, command.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// TODO: add record audit for not found user. But we should be careful to not leak information about which emails are registered in the system.
			return LoginResult{}, apperrors.ErrInvalidCredentials
		}

		return LoginResult{}, err
	}
	if err := s.passwordVerifier.Verify(user.PasswordHash, command.Password); err != nil {
		// TODO: add record audit for invalid password. But we should be careful to not leak information about which emails are registered in the system.
		return LoginResult{}, apperrors.ErrInvalidCredentials
	}

	membership, err := s.authRepository.FindMembership(ctx, user.ID, command.TenantID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.recordAudit(ctx, entity.AuditLog{EventType: "auth.login", Action: "login", Outcome: "invalid_membership", TenantID: &command.TenantID, UserID: &user.ID, Resource: "auth", Metadata: map[string]any{"email": command.Email}})
			return LoginResult{}, apperrors.ErrInvalidCredentials
		}

		return LoginResult{}, err
	}

	principal := entity.Principal{
		UserID:   user.ID,
		TenantID: membership.TenantID,
		Email:    user.Email,
		Role:     membership.Role,
	}

	accessToken, err := s.tokenIssuer.Issue(principal)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{AccessToken: accessToken, Principal: principal}, nil
}

func (s LoginService) recordAudit(ctx context.Context, entry entity.AuditLog) {
	if s.auditLogger != nil {
		_ = s.auditLogger.Record(ctx, entry)
	}
}
