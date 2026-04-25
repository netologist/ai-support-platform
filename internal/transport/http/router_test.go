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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	mockcommand "github.com/netologist/ai-support-platform/internal/mocks/command"
)

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

func TestNewRouter_LoginWiresExecutor(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	executor := mockcommand.NewMockLoginExecutor(t)
	var capturedCmd command.LoginCommand
	executor.EXPECT().Execute(mock.Anything, mock.Anything).
		Run(func(_ context.Context, cmd command.LoginCommand) {
			capturedCmd = cmd
		}).
		Return(command.LoginResult{
			AccessToken: "token.value",
			Principal: entity.Principal{
				UserID: userID,
				Email:  "user@example.com",
			},
		}, nil)
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

	assert.Equal(t, "user@example.com", capturedCmd.Email)
	assert.Equal(t, "secret", capturedCmd.Password)
	assert.Equal(t, nethttp.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "token.value")
	assert.Contains(t, rr.Body.String(), userID.String())
}
