package service

import (
	"errors"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

var ErrPermissionDenied = errors.New("permission denied")

type PasswordVerifier interface {
	Verify(hash string, plainText string) error
}

type TokenIssuer interface {
	Issue(principal entity.Principal) (string, error)
}

type TokenVerifier interface {
	Verify(token string) (entity.Principal, error)
}
