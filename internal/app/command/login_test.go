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
	fixedUserID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	fixedTenantID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	fixedToken := "signed.jwt.token"
	fixedUser := entity.User{
		ID:           fixedUserID,
		Email:        "user@example.com",
		PasswordHash: "$2a$hash",
	}
	fixedMembership := entity.Membership{
		UserID:   fixedUserID,
		TenantID: fixedTenantID,
		Role:     "agent",
	}
	fixedCmd := command.LoginCommand{
		Email:    "user@example.com",
		Password: "correct",
		TenantID: fixedTenantID,
	}
	fixedPrincipal := entity.Principal{
		UserID:   fixedUserID,
		TenantID: fixedTenantID,
		Email:    fixedUser.Email,
		Role:     fixedMembership.Role,
	}

	tests := []struct {
		name       string
		cmd        command.LoginCommand
		setupMocks func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditor *mocksvc.MockAuditLogger)
		wantToken  string
		wantErr    error
	}{
		{
			name: "valid credentials returns token and principal",
			cmd:  fixedCmd,
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditor *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, fixedUser.Email).Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, fixedCmd.Password).Return(nil)
				repo.EXPECT().FindMembership(mock.Anything, fixedUserID, fixedTenantID).Return(fixedMembership, nil)
				issuer.EXPECT().Issue(fixedPrincipal).Return(fixedToken, nil)
			},
			wantToken: fixedToken,
		},
		{
			name: "user not found returns ErrInvalidCredentials",
			cmd:  fixedCmd,
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditor *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, fixedUser.Email).Return(entity.User{}, repository.ErrNotFound)
			},
			wantErr: apperrors.ErrInvalidCredentials,
		},
		{
			name: "repository error on FindUserByEmail propagates",
			cmd:  fixedCmd,
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditor *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, fixedUser.Email).Return(entity.User{}, errors.New("db connection lost"))
			},
			wantErr: errors.New("db connection lost"),
		},
		{
			name: "wrong password returns ErrInvalidCredentials",
			cmd:  fixedCmd,
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditor *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, fixedUser.Email).Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, fixedCmd.Password).Return(errors.New("bcrypt mismatch"))
			},
			wantErr: apperrors.ErrInvalidCredentials,
		},
		{
			name: "membership not found returns ErrInvalidCredentials and records audit",
			cmd:  fixedCmd,
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditor *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, fixedUser.Email).Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, fixedCmd.Password).Return(nil)
				repo.EXPECT().FindMembership(mock.Anything, fixedUserID, fixedTenantID).Return(entity.Membership{}, repository.ErrNotFound)
				auditor.EXPECT().Record(mock.Anything, mock.MatchedBy(func(log entity.AuditLog) bool {
					return log.Outcome == "invalid_membership" && log.EventType == "auth.login"
				})).Return(nil)
			},
			wantErr: apperrors.ErrInvalidCredentials,
		},
		{
			name: "repository error on FindMembership propagates",
			cmd:  fixedCmd,
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditor *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, fixedUser.Email).Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, fixedCmd.Password).Return(nil)
				repo.EXPECT().FindMembership(mock.Anything, fixedUserID, fixedTenantID).Return(entity.Membership{}, errors.New("db timeout"))
			},
			wantErr: errors.New("db timeout"),
		},
		{
			name: "token issuer error propagates",
			cmd:  fixedCmd,
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditor *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, fixedUser.Email).Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, fixedCmd.Password).Return(nil)
				repo.EXPECT().FindMembership(mock.Anything, fixedUserID, fixedTenantID).Return(fixedMembership, nil)
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
			auditor := mocksvc.NewMockAuditLogger(t)

			tc.setupMocks(repo, verifier, issuer, auditor)

			svc := command.NewLoginService(repo, verifier, issuer, auditor)
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
