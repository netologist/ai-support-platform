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
	openapi_types "github.com/oapi-codegen/runtime/types"
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
	"github.com/netologist/ai-support-platform/internal/transport"
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
	return handlers.New(svc, nil, nil, nil, nil, nil, nil, 100, 100, time.Minute)
}

func newTicketHandler(
	t *testing.T,
	createExec *mockexec.MockCreateTicketExecutor,
	updateExec *mockexec.MockUpdateTicketExecutor,
	listExec *mockexec.MockListTicketsExecutor,
	getExec *mockexec.MockGetTicketExecutor,
) *handlers.Handlers {
	t.Helper()
	return handlers.New(nil, createExec, updateExec, listExec, getExec, nil, nil, 100, 100, time.Minute)
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
	return req.WithContext(transport.WithPrincipal(req.Context(), principal))
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
				auditLogger.EXPECT().Record(mock.Anything, mock.MatchedBy(func(log entity.AuditLog) bool {
					return log.Outcome == "user_not_found" && log.EventType == "auth.login"
				})).Return(nil)
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
				auditLogger.EXPECT().Record(mock.Anything, mock.MatchedBy(func(log entity.AuditLog) bool {
					return log.Outcome == "invalid_password" && log.EventType == "auth.login"
				})).Return(nil)
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

func TestCreateTicket(t *testing.T) {
	fixedTenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	fixedUserID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	fixedPrincipal := entity.Principal{UserID: fixedUserID, TenantID: fixedTenantID, Role: "agent"}
	createdTicket := entity.Ticket{ID: uuid.New(), TenantID: fixedTenantID, Subject: "Need assistance", Status: "open", CreatedByUserID: fixedUserID}

	tests := []struct {
		name       string
		principal  *entity.Principal
		body       string
		setupMocks func(exec *mockexec.MockCreateTicketExecutor)
		wantStatus int
		wantBody   string
	}{
		{
			name:      "success returns 201",
			principal: &fixedPrincipal,
			body:      `{"subject":"Need assistance"}`,
			setupMocks: func(exec *mockexec.MockCreateTicketExecutor) {
				exec.EXPECT().Execute(mock.Anything, command.CreateTicketCommand{Principal: fixedPrincipal, Subject: "Need assistance"}).Return(createdTicket, nil)
			},
			wantStatus: http.StatusCreated,
			wantBody:   "Need assistance",
		},
		{
			name:      "missing principal returns 401",
			principal: nil,
			body:      `{"subject":"Need assistance"}`,
			setupMocks: func(_ *mockexec.MockCreateTicketExecutor) {
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Unauthorized",
		},
		{
			name:      "bad json returns 400",
			principal: &fixedPrincipal,
			body:      `{bad-json}`,
			setupMocks: func(_ *mockexec.MockCreateTicketExecutor) {
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid request",
		},
		{
			name:      "forbidden returns 403",
			principal: &fixedPrincipal,
			body:      `{"subject":"Need assistance"}`,
			setupMocks: func(exec *mockexec.MockCreateTicketExecutor) {
				exec.EXPECT().Execute(mock.Anything, command.CreateTicketCommand{Principal: fixedPrincipal, Subject: "Need assistance"}).Return(entity.Ticket{}, apperrors.ErrForbidden)
			},
			wantStatus: http.StatusForbidden,
			wantBody:   "Forbidden",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			createExec := mockexec.NewMockCreateTicketExecutor(t)
			tc.setupMocks(createExec)

			h := newTicketHandler(t, createExec, nil, nil, nil)
			req := httptest.NewRequest(http.MethodPost, "/v1/tickets", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.principal != nil {
				req = requestWithPrincipal(req, *tc.principal)
			}

			rr := httptest.NewRecorder()
			h.CreateTicket(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.wantBody)
		})
	}
}

func TestUpdateTicket(t *testing.T) {
	fixedTenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	fixedUserID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	fixedTicketID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	fixedPrincipal := entity.Principal{UserID: fixedUserID, TenantID: fixedTenantID, Role: "agent"}
	updatedTicket := entity.Ticket{ID: fixedTicketID, TenantID: fixedTenantID, Subject: "Updated", Status: "closed", CreatedByUserID: fixedUserID}

	tests := []struct {
		name       string
		principal  *entity.Principal
		pathParam  string
		body       string
		setupMocks func(exec *mockexec.MockUpdateTicketExecutor)
		wantStatus int
		wantBody   string
	}{
		{
			name:      "success returns 200",
			principal: &fixedPrincipal,
			pathParam: fixedTicketID.String(),
			body:      `{"subject":" Updated ","status":"CLOSED"}`,
			setupMocks: func(exec *mockexec.MockUpdateTicketExecutor) {
				trimmed := "Updated"
				status := "closed"
				exec.EXPECT().Execute(mock.Anything, command.UpdateTicketCommand{
					TicketID:  fixedTicketID,
					Principal: fixedPrincipal,
					Subject:   &trimmed,
					Status:    &status,
				}).Return(updatedTicket, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "Updated",
		},
		{
			name:      "missing principal returns 401",
			principal: nil,
			pathParam: fixedTicketID.String(),
			body:      `{"subject":"Updated"}`,
			setupMocks: func(_ *mockexec.MockUpdateTicketExecutor) {
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Unauthorized",
		},
		{
			name:      "invalid ticket id returns 400",
			principal: &fixedPrincipal,
			pathParam: "not-a-uuid",
			body:      `{"subject":"Updated"}`,
			setupMocks: func(_ *mockexec.MockUpdateTicketExecutor) {
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Validation failed",
		},
		{
			name:      "not found returns 404",
			principal: &fixedPrincipal,
			pathParam: fixedTicketID.String(),
			body:      `{"subject":"Updated"}`,
			setupMocks: func(exec *mockexec.MockUpdateTicketExecutor) {
				subject := "Updated"
				exec.EXPECT().Execute(mock.Anything, command.UpdateTicketCommand{
					TicketID:  fixedTicketID,
					Principal: fixedPrincipal,
					Subject:   &subject,
					Status:    nil,
				}).Return(entity.Ticket{}, apperrors.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Not Found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			updateExec := mockexec.NewMockUpdateTicketExecutor(t)
			tc.setupMocks(updateExec)

			h := newTicketHandler(t, nil, updateExec, nil, nil)
			req := httptest.NewRequest(http.MethodPatch, "/v1/tickets/"+tc.pathParam, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = injectChiTicketID(req, tc.pathParam)
			if tc.principal != nil {
				req = requestWithPrincipal(req, *tc.principal)
			}

			rr := httptest.NewRecorder()
			h.UpdateTicket(rr, req, openapi_types.UUID(fixedTicketID))

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.wantBody)
		})
	}
}

func TestDocumentHandlers(t *testing.T) {
	fixedTenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	fixedUserID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	fixedPrincipal := entity.Principal{UserID: fixedUserID, TenantID: fixedTenantID, Role: "agent"}
	createdAt := time.Now().UTC()

	listExec := mockexec.NewMockListDocumentsExecutor(t)
	ingestExec := mockexec.NewMockIngestDocumentExecutor(t)

	doc := entity.KnowledgeDocument{
		ID:              uuid.New(),
		TenantID:        fixedTenantID,
		Title:           "Runbook",
		SourceURI:       "https://example.com/runbook",
		CreatedByUserID: fixedUserID,
		CreatedAt:       createdAt,
	}

	h := handlers.New(nil, nil, nil, nil, nil, ingestExec, listExec, 100, 100, time.Minute)

	t.Run("list documents missing principal returns 401", func(t *testing.T) {
		req := ticketRequest(t, http.MethodGet, "/v1/documents", nil)
		rr := httptest.NewRecorder()
		h.ListDocuments(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("list documents success", func(t *testing.T) {
		listExec.EXPECT().Execute(mock.Anything, query.ListDocumentsQuery{Principal: fixedPrincipal}).Return([]entity.KnowledgeDocument{doc}, nil).Once()
		req := requestWithPrincipal(ticketRequest(t, http.MethodGet, "/v1/documents", nil), fixedPrincipal)
		rr := httptest.NewRecorder()
		h.ListDocuments(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Runbook")
	})

	t.Run("ingest document success", func(t *testing.T) {
		ingestExec.EXPECT().Execute(mock.Anything, command.IngestDocumentCommand{
			Principal: fixedPrincipal,
			Title:     "Runbook",
			SourceURI: "https://example.com/runbook",
			Content:   "hello",
		}).Return(doc, nil).Once()

		req := requestWithPrincipal(ticketRequest(t, http.MethodPost, "/v1/documents", map[string]string{
			"title":      "Runbook",
			"source_uri": "https://example.com/runbook",
			"content":    "hello",
		}), fixedPrincipal)
		rr := httptest.NewRecorder()
		h.IngestDocument(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Contains(t, rr.Body.String(), "Runbook")
	})

	t.Run("ingest document malformed body returns 400", func(t *testing.T) {
		req := requestWithPrincipal(httptest.NewRequest(http.MethodPost, "/v1/documents", bytes.NewBufferString(`{bad-json}`)), fixedPrincipal)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.IngestDocument(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

// injectChiTicketID injects the ticketID URL param into the request context so that
// chi.URLParam(r, "ticketID") returns the expected value.
func injectChiTicketID(req *http.Request, ticketID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("ticketID", ticketID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
