package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/netologist/ai-support-platform/internal/app/executor"
	"github.com/netologist/ai-support-platform/internal/domain/service"
	"github.com/netologist/ai-support-platform/internal/transport"
	"github.com/netologist/ai-support-platform/internal/transport/http/handlers"
)

type Dependencies struct {
	LoginExecutor          executor.LoginExecutor
	CreateTicketExecutor   executor.CreateTicketExecutor
	UpdateTicketExecutor   executor.UpdateTicketExecutor
	ListTicketsExecutor    executor.ListTicketsExecutor
	GetTicketExecutor      executor.GetTicketExecutor
	ListDocumentsExecutor  executor.ListDocumentsExecutor
	IngestDocumentExecutor executor.IngestDocumentExecutor
	TokenVerifier          service.TokenVerifier
	RateLimiter            service.RateLimiter
	PublicRateLimit        int64
	AuthenticatedRateLimit int64
	RateLimitWindow        time.Duration
    GraphQLHandler         http.Handler
}

func NewRouter(dependencies Dependencies) http.Handler {
	swagger, err := handlers.GetSwagger()
	if err != nil {
		log.Fatal("failed to load swagger spec:", err)
	}
	swagger.Servers = nil

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.RequestSize(4 * 1024 * 1024)) // 4 MiB — guard against request body DoS
	router.Use(securityHeadersMiddleware)

	router.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/openapi.json"), // spec URL
	))

	router.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		swagger, _ := handlers.GetSwagger()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(swagger); err != nil {
			log.Printf("openapi.json encode error: %v", err)
		}
	})

	if dependencies.GraphQLHandler != nil {
		router.Group(func(router chi.Router) {
			router.Use(rateLimitMiddleware(
				dependencies.RateLimiter,
				dependencies.PublicRateLimit,
				dependencies.AuthenticatedRateLimit,
				dependencies.RateLimitWindow,
			))
			router.Use(authenticatedMiddlewareForGraphql(dependencies))
			router.Post("/graphql", dependencies.GraphQLHandler.ServeHTTP)
		})
	}

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
			dependencies.IngestDocumentExecutor,
			dependencies.ListDocumentsExecutor,
			handlers.HandlerConfig{
				PublicRateLimit:        dependencies.PublicRateLimit,
				AuthenticatedRateLimit: dependencies.AuthenticatedRateLimit,
				RateLimitWindow:        dependencies.RateLimitWindow,
			},
		), handlers.ChiServerOptions{
			BaseRouter: r,
			Middlewares: []handlers.MiddlewareFunc{
				rateLimitMiddleware(
					dependencies.RateLimiter,
					dependencies.PublicRateLimit,
					dependencies.AuthenticatedRateLimit,
					dependencies.RateLimitWindow,
				),
				authenticatedMiddleware(dependencies.TokenVerifier),
			},
		})
	})

	return router
}

var errUnauthorized = errors.New("unauthorized")

func authenticatedMiddlewareForGraphql(dependencies Dependencies) handlers.MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if dependencies.TokenVerifier == nil {
                writeUnauthorized(w, r)
                return
            }
            token, err := bearerTokenFromAuthorizationHeader(r.Header.Get("Authorization"))
            if err != nil {
                writeUnauthorized(w, r)
                return
            }
            principal, err := dependencies.TokenVerifier.Verify(token)
            if err != nil {
                writeUnauthorized(w, r)
                return
            }
            dependencies.GraphQLHandler.ServeHTTP(w, r.WithContext(transport.WithPrincipal(r.Context(), principal)))
        })
    }
}

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

			next.ServeHTTP(w, r.WithContext(transport.WithPrincipal(r.Context(), principal)))
		})
	}
}

func rateLimitMiddleware(
	limiter service.RateLimiter,
	publicLimit int64,
	authenticatedLimit int64,
	window time.Duration,
) handlers.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			key, limit := buildRateLimitKeyAndLimit(r, publicLimit, authenticatedLimit)
			result, err := limiter.Allow(r.Context(), key, limit, window)
			if err != nil {
				// fail-open tercih: limiter down olsa bile request devam etsin
				next.ServeHTTP(w, r)
				return
			}

			setRateLimitHeaders(w, result)

			if !result.Allowed {
				writeTooManyRequests(w, result.RetryAfter)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func setRateLimitHeaders(w http.ResponseWriter, result service.RateLimitResult) {
	remaining := result.Limit - result.Count
	if remaining < 0 {
		remaining = 0
	}

	w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
	w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
}

func writeTooManyRequests(w http.ResponseWriter, retryAfter time.Duration) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds()), 10))
	w.WriteHeader(http.StatusTooManyRequests)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "about:blank",
		"title":  "Too Many Requests",
		"status": http.StatusTooManyRequests,
		"detail": "rate limit exceeded",
	})
}

func buildRateLimitKeyAndLimit(
	r *http.Request,
	publicLimit int64,
	authenticatedLimit int64,
) (string, int64) {
	route := normalizeRoute(r.Method, r.URL.Path)

	principal, ok := transport.PrincipalFromContext(r.Context())
	if ok {
		key := fmt.Sprintf("auth:%s:%s:%s:%s",
			principal.TenantID.String(),
			principal.UserID.String(),
			r.Method,
			route,
		)
		return key, authenticatedLimit
	}

	ip := clientIP(r)
	key := fmt.Sprintf("public:%s:%s:%s", ip, r.Method, route)
	return key, publicLimit
}

func normalizeRoute(method, p string) string {
	// dinamik pathleri tek bucketta topla
	if strings.HasPrefix(p, "/v1/tickets/") {
		return "/v1/tickets/{ticketID}"
	}
	return p
}

func clientIP(r *http.Request) string {
	// X-Forwarded-For is only trustworthy when the service runs behind a
	// known reverse proxy. Trusting it blindly allows IP spoofing for rate-
	// limit bypass. Validate that the value is a parseable IP address before
	// using it; invalid/spoofed values fall back to RemoteAddr.
	xff := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	if xff != "" {
		if ip := net.ParseIP(xff); ip != nil {
			return ip.String()
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
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

// securityHeadersMiddleware adds defensive HTTP response headers on every
// request. These headers enable browser security features such as MIME-type
// enforcement and framing protection for API consumers that render responses.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}
