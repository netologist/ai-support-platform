package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

func TestTokenManagerIssueAndVerify(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-issuer", "secret-value", time.Hour)
	principal := entity.Principal{
		UserID:   uuid.New(),
		Email:    "agent@example.com",
	}

	token, err := manager.Issue(principal)
	require.NoError(t, err)

	verified, err := manager.Verify(token)
	require.NoError(t, err)
	require.Equal(t, principal.UserID, verified.UserID)
	require.Equal(t, principal.Email, verified.Email)
	require.False(t, verified.ExpiresAt.IsZero())
}

func TestTokenManagerVerifyRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-issuer", "secret-value", time.Hour)

	_, err := manager.Verify("not-a-token")
	require.ErrorIs(t, err, ErrInvalidToken)
}
