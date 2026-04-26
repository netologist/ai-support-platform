package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/netologist/ai-support-platform/internal/app/executor"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	"github.com/netologist/ai-support-platform/internal/transport/http/handlers"
)

type Dependencies struct {
	LoginExecutor        executor.LoginExecutor
	CreateTicketExecutor executor.CreateTicketExecutor
	UpdateTicketExecutor executor.UpdateTicketExecutor
	ListTicketsExecutor  executor.ListTicketsExecutor
	GetTicketExecutor    executor.GetTicketExecutor
	TokenVerifier        service.TokenVerifier
	Authorizer           service.Authorizer
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
		r.Use(nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, &nethttpmiddleware.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			},
		}))
		handlers.HandlerWithOptions(handlers.New(
			dependencies.LoginExecutor,
			dependencies.CreateTicketExecutor,
			dependencies.UpdateTicketExecutor,
			dependencies.ListTicketsExecutor,
			dependencies.GetTicketExecutor,
			100,               // publicRateLimit
			1000,              // authenticatedRateLimit
			15*60*time.Second, // rateLimitWindow
		), handlers.ChiServerOptions{
			BaseRouter: r,
			Middlewares: []handlers.MiddlewareFunc{
				authorizationGuardMiddleware(dependencies.Authorizer),
				authenticatedMiddleware(dependencies.TokenVerifier),
			},
		})
	})

	return router
}

type requiredPermission struct {
	resource string
	action   string
}

func authorizationGuardMiddleware(authorizer service.Authorizer) handlers.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			permission, requiresGuard := routePermission(r.Method, r.URL.Path)
			if !requiresGuard || authorizer == nil {
				next.ServeHTTP(w, r)
				return
			}

			principal, ok := handlers.PrincipalFromContext(r.Context())
			if !ok {
				writeUnauthorized(w, r)
				return
			}

			if err := authorizer.Authorize(r.Context(), principal, permission.resource, permission.action); err != nil {
				if errors.Is(err, service.ErrPermissionDenied) {
					writeForbidden(w)
					return
				}

				writeInternalServerError(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func routePermission(method string, requestPath string) (requiredPermission, bool) {
	switch {
	case method == http.MethodGet && requestPath == "/v1/tickets":
		return requiredPermission{resource: "tickets", action: "read"}, true
	case method == http.MethodPost && requestPath == "/v1/tickets":
		return requiredPermission{resource: "tickets", action: "create"}, true
	case (method == http.MethodGet || method == http.MethodPatch) && strings.HasPrefix(requestPath, "/v1/tickets/"):
		action := "read"
		if method == http.MethodPatch {
			action = "update"
		}

		if path.Base(requestPath) == "" || path.Base(requestPath) == "tickets" {
			return requiredPermission{}, false
		}

		return requiredPermission{resource: "tickets", action: action}, true
	default:
		return requiredPermission{}, false
	}
}

var errUnauthorized = errors.New("unauthorized")

func authenticatedMiddleware(tokenVerifier service.TokenVerifier) handlers.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, requiresAuth := r.Context().Value(handlers.BearerAuthScopes).([]string); !requiresAuth {
				next.ServeHTTP(w, r)
				return
			}

			if tokenVerifier == nil {
				writeUnauthorized(w, r)
				return
			}

			token, err := bearerTokenFromAuthorizationHeader(r.Header.Get("Authorization"))
			if err != nil {
				writeUnauthorized(w, r)
				return
			}

			principal, err := tokenVerifier.Verify(token)
			if err != nil {
				writeUnauthorized(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(handlers.WithPrincipal(r.Context(), principal)))
		})
	}
}

func bearerTokenFromAuthorizationHeader(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(strings.ToLower(header), strings.ToLower(prefix)) {
		return "", errUnauthorized
	}

	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", errUnauthorized
	}

	return token, nil
}

func writeUnauthorized(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "about:blank",
		"title":  "Unauthorized",
		"status": http.StatusUnauthorized,
		"detail": "missing or invalid bearer token",
	})
}

func writeForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "about:blank",
		"title":  "Forbidden",
		"status": http.StatusForbidden,
		"detail": "you are not allowed to access this resource",
	})
}

func writeInternalServerError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "about:blank",
		"title":  "Internal Server Error",
		"status": http.StatusInternalServerError,
		"detail": "unexpected error",
	})
}
