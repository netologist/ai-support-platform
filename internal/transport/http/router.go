package http

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	"github.com/netologist/ai-support-platform/internal/transport/http/handlers"
)

type Dependencies struct {
	LoginExecutor command.LoginExecutor
	TokenVerifier service.TokenVerifier
}

func NewRouter(dependencies Dependencies) http.Handler {
	swagger, err := handlers.GetSwagger()
	if err != nil {
		log.Fatal("swagger yüklenemedi:", err)
	}
	swagger.Servers = nil

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.Recoverer)

	router.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/openapi.json"), // spec URL
	))

	router.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		swagger, _ := handlers.GetSwagger()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swagger)
	})

	router.Group(func(r chi.Router) {
		r.Use(nethttpmiddleware.OapiRequestValidator(swagger))
		handlers.HandlerFromMux(handlers.New(
			dependencies.LoginExecutor,
			100,               // publicRateLimit
			1000,              // authenticatedRateLimit
			15*60*time.Second, // rateLimitWindow
		), r)
	})

	return router
}
