package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/app/command"
	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	mockexec "github.com/netologist/ai-support-platform/internal/mocks/executor"
	mockrepo "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mocksvc "github.com/netologist/ai-support-platform/internal/mocks/service"
	"github.com/netologist/ai-support-platform/internal/transport/http/handlers"
)

// --- helpers ---

func newLoginHandler(
	t *testing.T,
	repo *mockrepo.MockAuthRepository,
	verifier *mocksvc.MockPasswordVerifier,
	issuer *mocksvc.MockTokenIssuer,
	auditLogger *mocksvc.MockAuditLogger,
) *handlers.Handlers {
	t.Helper()
	svc := command.NewLoginService(repo, verifier, issuer, auditLogger)
	return handlers.New(svc, nil, nil, nil, nil, 100, 100, time.Minute)
}

func newTicketHandler(
	t *testing.T,
	createExec *mockexec.MockCreateTicketExecutor,
	updateExec *mockexec.MockUpdateTicketExecutor,
	listExec *mockexec.MockListTicketsExecutor,
	getExec *mockexec.MockGetTicketExecutor,
) *handlers.Handlers {
	t.Helper()
	return handlers.New(nil, createExec, updateExec, listExec, getExec, 100, 100, time.Minute)
}

func loginRequest(t *testing.T, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func requestWithPrincipal(req *http.Request, principal entity.Principal) *http.Request {
	return req.WithContext(handlers.WithPrincipal(req.Context(), principal))
}

func ticketRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	return req
}

// --- GetHealth ---

func TestGetHealth(t *testing.T) {
	h := newLoginHandler(t,
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

			h := newLoginHandler(t, repo, verifier, issuer, auditLogger)
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

	h := newLoginHandler(t, repo, verifier, issuer, auditLogger)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login",
		bytes.NewBufferString(`{"email":"USER@EXAMPLE.COM","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- ListTickets ---

func TestListTickets(t *testing.T) {
	fixedTenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	fixedUserID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	fixedPrincipal := entity.Principal{UserID: fixedUserID, TenantID: fixedTenantID, Role: "agent"}

	fixedTickets := []entity.Ticket{
		{ID: uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"), TenantID: fixedTenantID, Subject: "First ticket", Status: "open", CreatedByUserID: fixedUserID},
		{ID: uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"), TenantID: fixedTenantID, Subject: "Second ticket", Status: "closed", CreatedByUserID: fixedUserID},
	}

	tests := []struct {
		name       string
		principal  *entity.Principal
		setupMocks func(exec *mockexec.MockListTicketsExecutor)
		wantStatus int
		wantBody   string
	}{
		{
			name:      "authenticated user gets 200 with tickets",
			principal: &fixedPrincipal,
			setupMocks: func(exec *mockexec.MockListTicketsExecutor) {
				exec.EXPECT().Execute(mock.Anything, query.ListTicketsQuery{Principal: fixedPrincipal}).Return(fixedTickets, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "First ticket",
		},
		{
			name:      "returns empty array when no tickets",
			principal: &fixedPrincipal,
			setupMocks: func(exec *mockexec.MockListTicketsExecutor) {
				exec.EXPECT().Execute(mock.Anything, query.ListTicketsQuery{Principal: fixedPrincipal}).Return([]entity.Ticket{}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "[]",
		},
		{
			name:      "missing principal returns 401",
			principal: nil,
			setupMocks: func(_ *mockexec.MockListTicketsExecutor) {
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Unauthorized",
		},
		{
			name:      "forbidden returns 403",
			principal: &fixedPrincipal,
			setupMocks: func(exec *mockexec.MockListTicketsExecutor) {
				exec.EXPECT().Execute(mock.Anything, query.ListTicketsQuery{Principal: fixedPrincipal}).Return(nil, apperrors.ErrForbidden)
			},
			wantStatus: http.StatusForbidden,
			wantBody:   "Forbidden",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			listExec := mockexec.NewMockListTicketsExecutor(t)
			tc.setupMocks(listExec)

			h := newTicketHandler(t, nil, nil, listExec, nil)
			req := ticketRequest(t, http.MethodGet, "/v1/tickets", nil)
			if tc.principal != nil {
				req = requestWithPrincipal(req, *tc.principal)
			}
			rr := httptest.NewRecorder()
			h.ListTickets(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.wantBody)
		})
	}
}

// --- GetTicket ---

func TestGetTicket(t *testing.T) {
	fixedTenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	fixedUserID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	fixedTicketID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	fixedPrincipal := entity.Principal{UserID: fixedUserID, TenantID: fixedTenantID, Role: "agent"}
	fixedTicket := entity.Ticket{ID: fixedTicketID, TenantID: fixedTenantID, Subject: "Support request", Status: "open", CreatedByUserID: fixedUserID}

	fixedQuery := query.GetTicketQuery{
		TicketID:   fixedTicketID,
		Principal:  fixedPrincipal,
		Resource:   "tickets",
		ActionName: "read",
	}

	tests := []struct {
		name       string
		principal  *entity.Principal
		setupMocks func(exec *mockexec.MockGetTicketExecutor)
		wantStatus int
		wantBody   string
	}{
		{
			name:      "returns ticket for valid ID",
			principal: &fixedPrincipal,
			setupMocks: func(exec *mockexec.MockGetTicketExecutor) {
				exec.EXPECT().Execute(mock.Anything, fixedQuery).Return(fixedTicket, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "Support request",
		},
		{
			name:      "missing principal returns 401",
			principal: nil,
			setupMocks: func(_ *mockexec.MockGetTicketExecutor) {
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Unauthorized",
		},
		{
			name:      "ticket not found returns 404",
			principal: &fixedPrincipal,
			setupMocks: func(exec *mockexec.MockGetTicketExecutor) {
				exec.EXPECT().Execute(mock.Anything, fixedQuery).Return(entity.Ticket{}, apperrors.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Not Found",
		},
		{
			name:      "forbidden returns 403",
			principal: &fixedPrincipal,
			setupMocks: func(exec *mockexec.MockGetTicketExecutor) {
				exec.EXPECT().Execute(mock.Anything, fixedQuery).Return(entity.Ticket{}, apperrors.ErrForbidden)
			},
			wantStatus: http.StatusForbidden,
			wantBody:   "Forbidden",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getExec := mockexec.NewMockGetTicketExecutor(t)
			tc.setupMocks(getExec)

			h := newTicketHandler(t, nil, nil, nil, getExec)

			req := ticketRequest(t, http.MethodGet, "/v1/tickets/"+fixedTicketID.String(), nil)
			req = injectChiTicketID(req, fixedTicketID.String())
			if tc.principal != nil {
				req = requestWithPrincipal(req, *tc.principal)
			}
			rr := httptest.NewRecorder()
			h.GetTicket(rr, req, fixedTicketID)

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.wantBody)
		})
	}
}

// injectChiTicketID injects the ticketID URL param into the request context so that
// chi.URLParam(r, "ticketID") returns the expected value.
func injectChiTicketID(req *http.Request, ticketID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("ticketID", ticketID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
