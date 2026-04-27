package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

func TestTokenManagerIssueAndVerify(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-issuer", "secret-value", time.Hour)
	principal := entity.Principal{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Email:    "agent@example.com",
		Role:     "agent",
	}

	token, err := manager.Issue(principal)
	require.NoError(t, err)

	verified, err := manager.Verify(token)
	require.NoError(t, err)
	require.Equal(t, principal.UserID, verified.UserID)
	require.Equal(t, principal.TenantID, verified.TenantID)
	require.Equal(t, principal.Email, verified.Email)
	require.Equal(t, principal.Role, verified.Role)
	require.False(t, verified.ExpiresAt.IsZero())
}

func TestTokenManagerIssueAndVerify_AllRoles(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-issuer", "secret-value", time.Hour)

	for _, role := range []string{"admin", "agent", "viewer"} {
		role := role
		t.Run(role, func(t *testing.T) {
			t.Parallel()

			principal := entity.Principal{
				UserID:   uuid.New(),
				TenantID: uuid.New(),
				Email:    role + "@example.com",
				Role:     role,
			}

			token, err := manager.Issue(principal)
			require.NoError(t, err)

			verified, err := manager.Verify(token)
			require.NoError(t, err)
			require.Equal(t, principal.TenantID, verified.TenantID)
			require.Equal(t, role, verified.Role)
		})
	}
}

func TestTokenManagerVerifyRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-issuer", "secret-value", time.Hour)

	_, err := manager.Verify("not-a-token")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenManagerVerifyRejectsWrongAlgorithm(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-issuer", "secret-value", time.Hour)

	// RS256 ile imzalanmış bir token oluştur
	privateKey, err := generateTestRSAKey()
	require.NoError(t, err)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims{
		Email: "attacker@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: uuid.New().String(),
		},
	})
	signed, err := token.SignedString(privateKey)
	require.NoError(t, err)

	_, err = manager.Verify(signed)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenManagerVerifyRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-issuer", "secret-value", -time.Minute)
	principal := entity.Principal{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Email:    "user@example.com",
		Role:     "viewer",
	}

	token, err := manager.Issue(principal)
	require.NoError(t, err)

	_, err = manager.Verify(token)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenManagerVerifyRejectsTamperedToken(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-issuer", "secret-value", time.Hour)
	principal := entity.Principal{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Email:    "user@example.com",
		Role:     "viewer",
	}

	token, err := manager.Issue(principal)
	require.NoError(t, err)

	_, err = manager.Verify(token + "tampered")
	require.ErrorIs(t, err, ErrInvalidToken)
}
