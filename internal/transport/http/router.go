package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type Dependencies struct {
	LoginService           command.LoginService
	TokenVerifier          service.TokenVerifier
}

func NewHandler(dependencies Dependencies) http.Handler {
	handler := Handler{
		loginService:           dependencies.LoginService,
		tokenVerifier:          dependencies.TokenVerifier,
		validator:              NewRequestValidator(),
	}

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.Recoverer)

	router.Get("/healthz", handler.health)
	router.Get("/openapi.yaml", handler.openapiSpec)
	router.Get("/docs", handler.swaggerUI)
	router.Route("/v1", func(router chi.Router) {
		router.Post("/auth/login", handler.login)
	})

	return router
}
