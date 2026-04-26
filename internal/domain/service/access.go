package service

import (
	"context"
	"errors"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

var ErrPermissionDenied = errors.New("permission denied")

type Authorizer interface {
	Authorize(ctx context.Context, principal entity.Principal, resource string, action string) error
}

type PasswordVerifier interface {
	Verify(hash string, plainText string) error
}

type TokenIssuer interface {
	Issue(principal entity.Principal) (string, error)
}

type TokenVerifier interface {
	Verify(token string) (entity.Principal, error)
}
