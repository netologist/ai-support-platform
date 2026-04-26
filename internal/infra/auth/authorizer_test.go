package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	"github.com/netologist/ai-support-platform/internal/infra/auth"
)

func TestAuthorizerAuthorize(t *testing.T) {
	t.Parallel()

	authorizer, err := auth.NewAuthorizer()
	require.NoError(t, err)

	agentPrincipal := entity.Principal{UserID: uuid.New(), TenantID: uuid.New(), Role: "agent"}
	viewerPrincipal := entity.Principal{UserID: uuid.New(), TenantID: uuid.New(), Role: "viewer"}
	unknownPrincipal := entity.Principal{UserID: uuid.New(), TenantID: uuid.New(), Role: "unknown"}

	require.NoError(t, authorizer.Authorize(context.Background(), agentPrincipal, "tickets", "read"))
	require.NoError(t, authorizer.Authorize(context.Background(), viewerPrincipal, "tickets", "read"))
	require.ErrorIs(t, authorizer.Authorize(context.Background(), viewerPrincipal, "tickets", "create"), service.ErrPermissionDenied)
	require.ErrorIs(t, authorizer.Authorize(context.Background(), unknownPrincipal, "tickets", "read"), service.ErrPermissionDenied)
}
