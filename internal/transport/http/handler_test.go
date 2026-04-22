package http

import (
	"net/http"
	"testing"

	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

func TestHandler_health(t *testing.T) {
	type fields struct {
		loginService  command.LoginService
		tokenVerifier service.TokenVerifier
		validator     RequestValidator
	}
	type args struct {
		writer  http.ResponseWriter
		request *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Handler{
				loginService:  tt.fields.loginService,
				tokenVerifier: tt.fields.tokenVerifier,
				validator:     tt.fields.validator,
			}
			handler.health(tt.args.writer, tt.args.request)
		})
	}
}

func TestHandler_openapiSpec(t *testing.T) {
	type fields struct {
		loginService  command.LoginService
		tokenVerifier service.TokenVerifier
		validator     RequestValidator
	}
	type args struct {
		writer  http.ResponseWriter
		request *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Handler{
				loginService:  tt.fields.loginService,
				tokenVerifier: tt.fields.tokenVerifier,
				validator:     tt.fields.validator,
			}
			handler.openapiSpec(tt.args.writer, tt.args.request)
		})
	}
}

func TestHandler_swaggerUI(t *testing.T) {
	type fields struct {
		loginService  command.LoginService
		tokenVerifier service.TokenVerifier
		validator     RequestValidator
	}
	type args struct {
		writer  http.ResponseWriter
		request *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Handler{
				loginService:  tt.fields.loginService,
				tokenVerifier: tt.fields.tokenVerifier,
				validator:     tt.fields.validator,
			}
			handler.swaggerUI(tt.args.writer, tt.args.request)
		})
	}
}

func TestHandler_login(t *testing.T) {
	type fields struct {
		loginService  command.LoginService
		tokenVerifier service.TokenVerifier
		validator     RequestValidator
	}
	type args struct {
		writer  http.ResponseWriter
		request *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Handler{
				loginService:  tt.fields.loginService,
				tokenVerifier: tt.fields.tokenVerifier,
				validator:     tt.fields.validator,
			}
			handler.login(tt.args.writer, tt.args.request)
		})
	}
}
