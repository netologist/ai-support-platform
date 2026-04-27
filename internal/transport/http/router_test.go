package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/app/command"
	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	mockcommand "github.com/netologist/ai-support-platform/internal/mocks/command"
	mockexecutor "github.com/netologist/ai-support-platform/internal/mocks/executor"
	mockservice "github.com/netologist/ai-support-platform/internal/mocks/service"
)

type stubTokenVerifier struct {
	verifyFn func(token string) (entity.Principal, error)
}

func (stub stubTokenVerifier) Verify(token string) (entity.Principal, error) {
	if stub.verifyFn == nil {
		return entity.Principal{}, errors.New("verifyFn is nil")
	}

	return stub.verifyFn(token)
}

// loginPayload builds a valid login JSON body as required by the OpenAPI schema.
func loginPayload(t *testing.T, email, password string, tenantID uuid.UUID) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(map[string]string{
		"email":     email,
		"password":  password,
		"tenant_id": tenantID.String(),
	})
	require.NoError(t, err)
	return bytes.NewReader(b)
}

func TestNewRouter_Healthz(t *testing.T) {
	t.Parallel()

	executor := mockcommand.NewMockLoginExecutor(t)
	router := NewRouter(Dependencies{LoginExecutor: executor})
	req := httptest.NewRequest(nethttp.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"ok"`)
}

func TestNewRouter_OpenAPIJSON(t *testing.T) {
	t.Parallel()

	executor := mockcommand.NewMockLoginExecutor(t)
	router := NewRouter(Dependencies{LoginExecutor: executor})
	req := httptest.NewRequest(nethttp.MethodGet, "/openapi.json", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, rr.Body.String(), `"openapi"`)
}

func TestNewRouter_UnknownRoute(t *testing.T) {
	t.Parallel()

	executor := mockcommand.NewMockLoginExecutor(t)
	router := NewRouter(Dependencies{LoginExecutor: executor})
	req := httptest.NewRequest(nethttp.MethodGet, "/does/not/exist", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusNotFound, rr.Code)
}

func TestNewRouter_LoginWiresExecutor(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	tenantID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	executor := mockcommand.NewMockLoginExecutor(t)
	var capturedCmd command.LoginCommand
	executor.EXPECT().Execute(mock.Anything, mock.Anything).
		Run(func(_ context.Context, cmd command.LoginCommand) {
			capturedCmd = cmd
		}).
		Return(command.LoginResult{
			AccessToken: "token.value",
			Principal: entity.Principal{
				UserID:   userID,
				TenantID: tenantID,
				Email:    "user@example.com",
			},
		}, nil)

	router := NewRouter(Dependencies{LoginExecutor: executor})
	req := httptest.NewRequest(nethttp.MethodPost, "/v1/auth/login",
		loginPayload(t, "USER@example.com", "secret", tenantID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusOK, rr.Code)
	assert.Equal(t, "user@example.com", capturedCmd.Email)
	assert.Equal(t, "secret", capturedCmd.Password)
	assert.Contains(t, rr.Body.String(), "token.value")
	assert.Contains(t, rr.Body.String(), userID.String())
}

func TestNewRouter_LoginInvalidCredentials(t *testing.T) {
	t.Parallel()

	tenantID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	executor := mockcommand.NewMockLoginExecutor(t)
	executor.EXPECT().Execute(mock.Anything, mock.Anything).Return(command.LoginResult{}, apperrors.ErrInvalidCredentials)

	router := NewRouter(Dependencies{LoginExecutor: executor})
	req := httptest.NewRequest(nethttp.MethodPost, "/v1/auth/login",
		loginPayload(t, "user@example.com", "wrong", tenantID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Unauthorized")
}

func TestNewRouter_LoginMissingTenantIDRejectedByValidator(t *testing.T) {
	t.Parallel()

	executor := mockcommand.NewMockLoginExecutor(t)
	router := NewRouter(Dependencies{LoginExecutor: executor})

	b, err := json.Marshal(map[string]string{"email": "user@example.com", "password": "secret"})
	require.NoError(t, err)
	req := httptest.NewRequest(nethttp.MethodPost, "/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// OpenAPI validator rejects the request before it reaches the handler
	assert.Equal(t, nethttp.StatusBadRequest, rr.Code)
}

func TestNewRouter_TicketsRequiresAuthentication(t *testing.T) {
	t.Parallel()

	loginExecutor := mockcommand.NewMockLoginExecutor(t)
	listExecutor := mockexecutor.NewMockListTicketsExecutor(t)

	router := NewRouter(Dependencies{
		LoginExecutor:       loginExecutor,
		ListTicketsExecutor: listExecutor,
	})

	req := httptest.NewRequest(nethttp.MethodGet, "/v1/tickets", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Unauthorized")
}

func TestNewRouter_TicketsRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	loginExecutor := mockcommand.NewMockLoginExecutor(t)
	listExecutor := mockexecutor.NewMockListTicketsExecutor(t)

	router := NewRouter(Dependencies{
		LoginExecutor:       loginExecutor,
		ListTicketsExecutor: listExecutor,
		TokenVerifier: stubTokenVerifier{
			verifyFn: func(_ string) (entity.Principal, error) {
				return entity.Principal{}, errors.New("invalid token")
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodGet, "/v1/tickets", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Unauthorized")
}

func TestNewRouter_TicketsAllowsValidToken(t *testing.T) {
	t.Parallel()

	loginExecutor := mockcommand.NewMockLoginExecutor(t)
	listExecutor := mockexecutor.NewMockListTicketsExecutor(t)

	expectedPrincipal := entity.Principal{
		UserID:   uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		TenantID: uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		Role:     "agent",
	}

	listExecutor.EXPECT().Execute(mock.Anything, mock.MatchedBy(func(q query.ListTicketsQuery) bool {
		return q.Principal.UserID == expectedPrincipal.UserID &&
			q.Principal.TenantID == expectedPrincipal.TenantID &&
			q.Principal.Role == expectedPrincipal.Role
	})).Return([]entity.Ticket{}, nil)

	router := NewRouter(Dependencies{
		LoginExecutor:       loginExecutor,
		ListTicketsExecutor: listExecutor,
		TokenVerifier: stubTokenVerifier{
			verifyFn: func(token string) (entity.Principal, error) {
				if token != "valid-token" {
					return entity.Principal{}, errors.New("invalid token")
				}

				return expectedPrincipal, nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodGet, "/v1/tickets", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusOK, rr.Code)
}

func TestNewRouter_RateLimitExceededReturns429(t *testing.T) {
	t.Parallel()

	loginExecutor := mockcommand.NewMockLoginExecutor(t)
	listExecutor := mockexecutor.NewMockListTicketsExecutor(t)
	rateLimiter := mockservice.NewMockRateLimiter(t)

	principal := entity.Principal{
		UserID:   uuid.MustParse("80808080-8080-8080-8080-808080808080"),
		TenantID: uuid.MustParse("90909090-9090-9090-9090-909090909090"),
		Role:     "agent",
	}

	rateLimiter.EXPECT().Allow(mock.Anything, mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "auth:")
	}), int64(2), time.Minute).Return(service.RateLimitResult{
		Allowed:    false,
		Count:      3,
		Limit:      2,
		RetryAfter: 30 * time.Second,
	}, nil)

	router := NewRouter(Dependencies{
		LoginExecutor:          loginExecutor,
		ListTicketsExecutor:    listExecutor,
		TokenVerifier:          stubTokenVerifier{verifyFn: func(_ string) (entity.Principal, error) { return principal, nil }},
		RateLimiter:            rateLimiter,
		PublicRateLimit:        1,
		AuthenticatedRateLimit: 2,
		RateLimitWindow:        time.Minute,
	})

	req := httptest.NewRequest(nethttp.MethodGet, "/v1/tickets", nil)
	req.Header.Set("Authorization", "Bearer valid")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusTooManyRequests, rr.Code)
	assert.Contains(t, rr.Body.String(), "Too Many Requests")
	assert.Equal(t, "2", rr.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rr.Header().Get("X-RateLimit-Remaining"))
}
