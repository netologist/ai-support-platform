package http

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestValidatorDecodeAndValidate(t *testing.T) {
	t.Parallel()

	validator := NewRequestValidator()
	request := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(`{"email":"bad","password":""}`))

	var body loginRequest
	validationErrors, err := validator.DecodeAndValidate(request, &body)
	require.Error(t, err)
	require.Contains(t, validationErrors, "email")
	require.Contains(t, validationErrors, "password")
}
