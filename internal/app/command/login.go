package command

import (
	"context"
	"errors"

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
}

type LoginResult struct {
	AccessToken string
	Principal   entity.Principal
}

type LoginService struct {
	authRepository   repository.AuthRepository
	passwordVerifier service.PasswordVerifier
	tokenIssuer      service.TokenIssuer
}

func NewLoginService(
	authRepository repository.AuthRepository,
	passwordVerifier service.PasswordVerifier,
	tokenIssuer service.TokenIssuer,
) *LoginService {
	return &LoginService{
		authRepository:   authRepository,
		passwordVerifier: passwordVerifier,
		tokenIssuer:      tokenIssuer,
	}
}

func (service *LoginService) Execute(ctx context.Context, command LoginCommand) (LoginResult, error) {
	user, err := service.authRepository.FindUserByEmail(ctx, command.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// TODO: add record audit for not found user. But we should be careful to not leak information about which emails are registered in the system.
			return LoginResult{}, apperrors.ErrInvalidCredentials
		}

		return LoginResult{}, err
	}
	if err := service.passwordVerifier.Verify(user.PasswordHash, command.Password); err != nil {
		// TODO: add record audit for invalid password. But we should be careful to not leak information about which emails are registered in the system.
		return LoginResult{}, apperrors.ErrInvalidCredentials
	}

	principal := entity.Principal{
		UserID: user.ID,
		Email:  user.Email,
	}

	accessToken, err := service.tokenIssuer.Issue(principal)
	if err != nil {
		return LoginResult{}, err
	}

	// TODO: add record audit for successful login

	return LoginResult{AccessToken: accessToken, Principal: principal}, nil
}
