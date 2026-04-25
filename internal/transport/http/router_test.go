package http

import (
	"bytes"
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type stubLoginExecutor struct {
	called bool
	cmd    command.LoginCommand
	result command.LoginResult
	err    error
}

func (s *stubLoginExecutor) Execute(_ context.Context, cmd command.LoginCommand) (command.LoginResult, error) {
	s.called = true
	s.cmd = cmd
	return s.result, s.err
}

func TestNewRouter_Healthz(t *testing.T) {
	t.Parallel()

	router := NewRouter(Dependencies{LoginExecutor: &stubLoginExecutor{}})
	req := httptest.NewRequest(nethttp.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"ok"`)
}

func TestNewRouter_OpenAPIJSON(t *testing.T) {
	t.Parallel()

	router := NewRouter(Dependencies{LoginExecutor: &stubLoginExecutor{}})
	req := httptest.NewRequest(nethttp.MethodGet, "/openapi.json", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, nethttp.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, rr.Body.String(), `"openapi"`)
}

func TestNewRouter_LoginWiresExecutor(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	executor := &stubLoginExecutor{
		result: command.LoginResult{
			AccessToken: "token.value",
			Principal: entity.Principal{
				UserID: userID,
				Email:  "user@example.com",
			},
		},
	}
	router := NewRouter(Dependencies{LoginExecutor: executor})

	payload, err := json.Marshal(map[string]string{
		"email":    "USER@example.com",
		"password": "secret",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(nethttp.MethodPost, "/v1/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.True(t, executor.called)
	assert.Equal(t, "user@example.com", executor.cmd.Email)
	assert.Equal(t, "secret", executor.cmd.Password)
	assert.Equal(t, nethttp.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "token.value")
	assert.Contains(t, rr.Body.String(), userID.String())
}
