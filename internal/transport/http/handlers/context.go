package handlers

import (
	"context"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type contextKey string

const principalContextKey contextKey = "principal"

func withPrincipal(ctx context.Context, principal entity.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func principalFromContext(ctx context.Context) (entity.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(entity.Principal)
	return principal, ok
}

func WithPrincipal(ctx context.Context, principal entity.Principal) context.Context {
	return withPrincipal(ctx, principal)
}

func PrincipalFromContext(ctx context.Context) (entity.Principal, bool) {
	return principalFromContext(ctx)
}
