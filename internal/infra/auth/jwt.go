package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenManager struct {
	issuer string
	secret []byte
	ttl    time.Duration
}

type claims struct {
    TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
    jwt.RegisteredClaims
}

func NewTokenManager(issuer string, secret string, ttl time.Duration) TokenManager {
	return TokenManager{issuer: issuer, secret: []byte(secret), ttl: ttl}
}

func (manager TokenManager) Issue(principal entity.Principal) (string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(manager.ttl)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		TenantID: principal.TenantID.String(),
		Role:     principal.Role,
		Email:    principal.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    manager.issuer,
			Subject:   principal.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	return token.SignedString(manager.secret)
}

func (manager TokenManager) Verify(rawToken string) (entity.Principal, error) {
	parsed, err := jwt.ParseWithClaims(rawToken, &claims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidToken
		}

		return manager.secret, nil
	})
	if err != nil {
		return entity.Principal{}, ErrInvalidToken
	}

	tokenClaims, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return entity.Principal{}, ErrInvalidToken
	}

	userID, err := uuid.Parse(tokenClaims.Subject)
	if err != nil {
		return entity.Principal{}, ErrInvalidToken
	}

	tenantID, err := uuid.Parse(tokenClaims.TenantID)
	if err != nil {
		return entity.Principal{}, ErrInvalidToken
	}

	expiresAt := time.Time{}
	if tokenClaims.ExpiresAt != nil {
		expiresAt = tokenClaims.ExpiresAt.Time
	}

	return entity.Principal{
		UserID:    userID,
		TenantID:  tenantID,
		Email:     tokenClaims.Email,
		Role:      tokenClaims.Role,
		ExpiresAt: expiresAt,
	}, nil
}
