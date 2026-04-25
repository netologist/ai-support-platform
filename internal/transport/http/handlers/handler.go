package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/netologist/ai-support-platform/internal/app/command"
	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
)

var _ ServerInterface = (*Handlers)(nil)

func New(
	loginExecutor command.LoginExecutor,
	publicRateLimit int64,
	authenticatedRateLimit int64,
	rateLimitWindow time.Duration) *Handlers {
	return &Handlers{
		loginExecutor:          loginExecutor,
		publicRateLimit:        publicRateLimit,
		authenticatedRateLimit: authenticatedRateLimit,
		rateLimitWindow:        rateLimitWindow,
	}
}

type Handlers struct {
	loginExecutor          command.LoginExecutor
	publicRateLimit        int64
	authenticatedRateLimit int64
	rateLimitWindow        time.Duration
	// validator              RequestValidator
}

func (h *Handlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var body LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Invalid request", fmt.Sprintf("decode request body: %v", err), nil)
		return
	}

	result, err := h.loginExecutor.Execute(r.Context(), command.LoginCommand{
		Email:    strings.ToLower(string(body.Email)),
		Password: body.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidCredentials):
			writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "email or password is invalid", nil)
		default:
			writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "unexpected error", nil)
		}

		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		UserId:      result.Principal.UserID,
	})
}
