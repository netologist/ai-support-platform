package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/app/command"
	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	mockrepo "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mocksvc "github.com/netologist/ai-support-platform/internal/mocks/service"
)

func TestLoginService_Execute(t *testing.T) {
	fixedID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	fixedToken := "signed.jwt.token"
	fixedUser := entity.User{
		ID:           fixedID,
		Email:        "user@example.com",
		PasswordHash: "$2a$hash",
	}
	fixedPrincipal := entity.Principal{UserID: fixedID, Email: fixedUser.Email}

	tests := []struct {
		name       string
		cmd        command.LoginCommand
		setupMocks func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer)
		wantToken  string
		wantErr    error
	}{
		{
			name: "valid credentials returns token and principal",
			cmd:  command.LoginCommand{Email: "user@example.com", Password: "correct"},
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer) {
				repo.EXPECT().FindUserByEmail(mock.Anything, "user@example.com").Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, "correct").Return(nil)
				issuer.EXPECT().Issue(fixedPrincipal).Return(fixedToken, nil)
			},
			wantToken: fixedToken,
		},
		{
			name: "user not found returns ErrInvalidCredentials",
			cmd:  command.LoginCommand{Email: "ghost@example.com", Password: "pass"},
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer) {
				repo.EXPECT().FindUserByEmail(mock.Anything, "ghost@example.com").Return(entity.User{}, repository.ErrNotFound)
			},
			wantErr: apperrors.ErrInvalidCredentials,
		},
		{
			name: "wrong password returns ErrInvalidCredentials",
			cmd:  command.LoginCommand{Email: "user@example.com", Password: "wrong"},
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer) {
				repo.EXPECT().FindUserByEmail(mock.Anything, "user@example.com").Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, "wrong").Return(errors.New("bcrypt mismatch"))
			},
			wantErr: apperrors.ErrInvalidCredentials,
		},
		{
			name: "repository error propagates",
			cmd:  command.LoginCommand{Email: "user@example.com", Password: "pass"},
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer) {
				repo.EXPECT().FindUserByEmail(mock.Anything, "user@example.com").Return(entity.User{}, errors.New("db connection lost"))
			},
			wantErr: errors.New("db connection lost"),
		},
		{
			name: "token issuer error propagates",
			cmd:  command.LoginCommand{Email: "user@example.com", Password: "correct"},
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer) {
				repo.EXPECT().FindUserByEmail(mock.Anything, "user@example.com").Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, "correct").Return(nil)
				issuer.EXPECT().Issue(fixedPrincipal).Return("", errors.New("signing key unavailable"))
			},
			wantErr: errors.New("signing key unavailable"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mockrepo.NewMockAuthRepository(t)
			verifier := mocksvc.NewMockPasswordVerifier(t)
			issuer := mocksvc.NewMockTokenIssuer(t)

			tc.setupMocks(repo, verifier, issuer)

			svc := command.NewLoginService(repo, verifier, issuer)
			result, err := svc.Execute(context.Background(), tc.cmd)

			if tc.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tc.wantErr, apperrors.ErrInvalidCredentials) {
					assert.ErrorIs(t, err, apperrors.ErrInvalidCredentials)
				} else {
					assert.EqualError(t, err, tc.wantErr.Error())
				}
				assert.Empty(t, result.AccessToken)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantToken, result.AccessToken)
			assert.Equal(t, fixedPrincipal, result.Principal)
		})
	}
}
