package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/netologist/ai-support-platform/internal/app/command"
	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	"github.com/netologist/ai-support-platform/openapi"
)

type Handler struct {
	loginService           command.LoginService
    tokenVerifier          service.TokenVerifier
	validator              RequestValidator
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	UserID      string `json:"user_id"`
}


func (handler Handler) health(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (handler Handler) openapiSpec(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/yaml")
	_, _ = writer.Write(openapi.Spec)
}

func (handler Handler) swaggerUI(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write([]byte(`<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>AI Support Platform API Docs</title>
		<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
	</head>
	<body>
		<div id="swagger-ui"></div>
		<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
		<script>
			SwaggerUIBundle({ url: '/openapi.yaml', dom_id: '#swagger-ui' });
		</script>
	</body>
</html>`))
}

func (handler Handler) login(writer http.ResponseWriter, request *http.Request) {
	var body loginRequest
	validationErrors, err := handler.validator.DecodeAndValidate(request, &body)
	if err != nil {
		if len(validationErrors) > 0 {
			writeProblem(writer, request, http.StatusBadRequest, "Validation failed", "request body is invalid", validationErrors)
			return
		}

		writeProblem(writer, request, http.StatusBadRequest, "Validation failed", err.Error(), nil)
		return
	}

	result, err := handler.loginService.Execute(request.Context(), command.LoginCommand{
		Email:    strings.ToLower(body.Email),
		Password: body.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidCredentials):
			writeProblem(writer, request, http.StatusUnauthorized, "Unauthorized", "email, password, is invalid", nil)
		default:
			writeProblem(writer, request, http.StatusInternalServerError, "Internal Server Error", "unexpected error", nil)
		}

		return
	}

	writeJSON(writer, http.StatusOK, loginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		UserID:      result.Principal.UserID.String(),
	})
}
