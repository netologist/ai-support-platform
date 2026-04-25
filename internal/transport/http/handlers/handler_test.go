package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	"github.com/netologist/ai-support-platform/internal/transport/http/handlers"
)

// --- helpers ---

func newHandler(
	repo *mockrepo.MockAuthRepository,
	verifier *mocksvc.MockPasswordVerifier,
	issuer *mocksvc.MockTokenIssuer,
    auditLogger *mocksvc.MockAuditLogger,
) *handlers.Handlers {
	svc := command.NewLoginService(repo, verifier, issuer, auditLogger)
	return handlers.New(svc, 100, 100, time.Minute)
}

func loginRequest(t *testing.T, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// --- GetHealth ---

func TestGetHealth(t *testing.T) {
	h := newHandler(
		mockrepo.NewMockAuthRepository(t),
		mocksvc.NewMockPasswordVerifier(t),
		mocksvc.NewMockTokenIssuer(t),
		mocksvc.NewMockAuditLogger(t),
	)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	h.GetHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"ok"`)
}

// --- Login ---

func TestLogin(t *testing.T) {
	fixedID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fixedTenantID := uuid.Nil // handler does not pass TenantID; zero UUID flows through
	fixedMembership := entity.Membership{UserID: fixedID, TenantID: fixedTenantID, Role: "agent"}
	fixedToken := "test.jwt.token"
	fixedUser := entity.User{ID: fixedID, Email: "user@example.com", PasswordHash: "hash"}

	tests := []struct {
		name       string
		buildReq   func(t *testing.T) *http.Request
		setupMocks func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditLogger *mocksvc.MockAuditLogger)
		wantStatus int
		wantBody   string
	}{
		{
			name: "valid credentials returns 200 with token",
			buildReq: func(t *testing.T) *http.Request {
				return loginRequest(t, map[string]string{"email": "user@example.com", "password": "secret"})
			},
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditLogger *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, "user@example.com").Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, "secret").Return(nil)
				repo.EXPECT().FindMembership(mock.Anything, fixedID, fixedTenantID).Return(fixedMembership, nil)
				issuer.EXPECT().Issue(entity.Principal{UserID: fixedID, TenantID: fixedTenantID, Email: fixedUser.Email, Role: fixedMembership.Role}).Return(fixedToken, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   fixedToken,
		},
		{
			name: "user not found returns 401",
			buildReq: func(t *testing.T) *http.Request {
				return loginRequest(t, map[string]string{"email": "nobody@example.com", "password": "pass"})
			},
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditLogger *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, "nobody@example.com").Return(entity.User{}, repository.ErrNotFound)
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Unauthorized",
		},
		{
			name: "wrong password returns 401",
			buildReq: func(t *testing.T) *http.Request {
				return loginRequest(t, map[string]string{"email": "user@example.com", "password": "wrong"})
			},
			setupMocks: func(repo *mockrepo.MockAuthRepository, verifier *mocksvc.MockPasswordVerifier, issuer *mocksvc.MockTokenIssuer, auditLogger *mocksvc.MockAuditLogger) {
				repo.EXPECT().FindUserByEmail(mock.Anything, "user@example.com").Return(fixedUser, nil)
				verifier.EXPECT().Verify(fixedUser.PasswordHash, "wrong").Return(apperrors.ErrInvalidCredentials)
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Unauthorized",
		},
		{
			name: "malformed JSON returns 400",
			buildReq: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{not json}`))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			setupMocks: func(_ *mockrepo.MockAuthRepository, _ *mocksvc.MockPasswordVerifier, _ *mocksvc.MockTokenIssuer, _ *mocksvc.MockAuditLogger) {
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid request",
		},
		{
			name: "unknown JSON field returns 400",
			buildReq: func(t *testing.T) *http.Request {
				return loginRequest(t, map[string]string{"email": "user@example.com", "password": "secret", "extra": "field"})
			},
			setupMocks: func(_ *mockrepo.MockAuthRepository, _ *mocksvc.MockPasswordVerifier, _ *mocksvc.MockTokenIssuer, _ *mocksvc.MockAuditLogger) {
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid request",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mockrepo.NewMockAuthRepository(t)
			verifier := mocksvc.NewMockPasswordVerifier(t)
			issuer := mocksvc.NewMockTokenIssuer(t)
            auditLogger := mocksvc.NewMockAuditLogger(t)

			tc.setupMocks(repo, verifier, issuer, auditLogger)

			h := newHandler(repo, verifier, issuer, auditLogger)
			rr := httptest.NewRecorder()
			h.Login(rr, tc.buildReq(t))

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.wantBody)
		})
	}
}

func TestLogin_EmailNormalized(t *testing.T) {
	fixedID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	fixedTenantID := uuid.Nil
	fixedUser := entity.User{ID: fixedID, Email: "user@example.com", PasswordHash: "hash"}
	fixedMembership := entity.Membership{UserID: fixedID, TenantID: fixedTenantID, Role: "agent"}

	repo := mockrepo.NewMockAuthRepository(t)
	verifier := mocksvc.NewMockPasswordVerifier(t)
	issuer := mocksvc.NewMockTokenIssuer(t)
	auditLogger := mocksvc.NewMockAuditLogger(t)

	// Expect the lowercased email
	repo.EXPECT().FindUserByEmail(mock.Anything, "user@example.com").Return(fixedUser, nil)
	verifier.EXPECT().Verify(fixedUser.PasswordHash, "secret").Return(nil)
	repo.EXPECT().FindMembership(mock.Anything, fixedID, fixedTenantID).Return(fixedMembership, nil)
	issuer.EXPECT().Issue(entity.Principal{UserID: fixedID, TenantID: fixedTenantID, Email: fixedUser.Email, Role: fixedMembership.Role}).Return("tok", nil)

	h := newHandler(repo, verifier, issuer, auditLogger)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login",
		bytes.NewBufferString(`{"email":"USER@EXAMPLE.COM","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
