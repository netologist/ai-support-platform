package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/infra/ai/chunking"
	infraaudit "github.com/netologist/ai-support-platform/internal/infra/audit"
	infraauth "github.com/netologist/ai-support-platform/internal/infra/auth"
	infracache "github.com/netologist/ai-support-platform/internal/infra/cache"
	postgresrepo "github.com/netologist/ai-support-platform/internal/infra/repository/postgres"
	"github.com/netologist/ai-support-platform/internal/infra/repository/sqlc"
	transportgraphql "github.com/netologist/ai-support-platform/internal/transport/graphql"
	transporthttp "github.com/netologist/ai-support-platform/internal/transport/http"
)

type APIRuntime struct {
	Server *http.Server
	Close  func()
}

func NewAPIRuntime(ctx context.Context, config Config) (APIRuntime, error) {
	closer := &closerStack{}

	db, err := buildDB(ctx, config, closer)
	if err != nil {
		closer.Close()
		return APIRuntime{}, err
	}

	redisClient, err := buildRedis(ctx, config, closer)
	if err != nil {
		closer.Close()
		return APIRuntime{}, err
	}

	kafkaPublisher, err := buildKafkaPublisher(config, closer)
	if err != nil {
		closer.Close()
		return APIRuntime{}, err
	}

	aiProviders, err := buildAI(ctx, config, closer)
	if err != nil {
		closer.Close()
		return APIRuntime{}, err
	}

	// -------------------------
	// Repositories
	// -------------------------
	queries := sqlc.New(db)
	authRepository := postgresrepo.NewAuthRepository(queries)
	auditRepository := postgresrepo.NewAuditRepository(queries)
	ticketRepository := postgresrepo.NewTicketRepository(queries)
	documentRepository := postgresrepo.NewDocumentRepository(queries)
	chunkRepository := postgresrepo.NewChunkRepository(queries)
	outboxRepository := postgresrepo.NewOutboxRepository(queries)

	// -------------------------
	// Auth
	// -------------------------
	tokenManager := infraauth.NewTokenManager(config.JWTIssuer, config.JWTSecret, config.JWTTTL)

	// -------------------------
	// Services
	// -------------------------
	auditLogger := infraaudit.NewLogger(auditRepository, kafkaPublisher, config.KafkaAuditTopic)
	authorizer, err := infraauth.NewAuthorizer()
	if err != nil {
		closer.Close()
		return APIRuntime{}, fmt.Errorf("build authorizer: %w", err)
	}

	ticketCache := infracache.NewRedisTicketCache(redisClient, config.TicketCacheTTL)
	chunker := chunking.NewSimpleChunker()
	redisRateLimiter := infracache.NewRedisRateLimiter(redisClient)

	loginService := command.NewLoginService(
		authRepository,
		infraauth.PasswordVerifier{},
		tokenManager,
		auditLogger,
	)

	createTicketService := command.NewCreateTicketService(
		ticketRepository,
		authorizer,
		ticketCache,
		auditLogger,
		outboxRepository,
	)

	updateTicketService := command.NewUpdateTicketService(
		ticketRepository,
		authorizer,
		ticketCache,
		auditLogger,
		outboxRepository,
	)

	getTicketService := query.NewGetTicketService(
		ticketRepository,
		authorizer,
		ticketCache,
		auditLogger,
	)

	listTicketService := query.NewListTicketsService(
		ticketRepository,
		authorizer,
		auditLogger,
	)

	ingestDocumentService := command.NewIngestDocumentService(
		documentRepository,
		chunkRepository,
		chunker,
		aiProviders.EmbeddingProvider,
		authorizer,
		auditLogger,
	)

	listDocumentsService := query.NewListDocumentsService(
		documentRepository,
		authorizer,
		auditLogger,
	)

	// -------------------------
	// GraphQL
	// -------------------------
	graphQLHandler := transportgraphql.NewHandler(transportgraphql.Dependencies{
		GetTicketService:      getTicketService,
		ListTicketsService:    listTicketService,
		ListDocumentsService:  listDocumentsService,
		// SemanticSearchService: semanticSearchService,
		// SuggestReplyService:   suggestReplyService,
	})

	// -------------------------
	// HTTP
	// -------------------------
	router := transporthttp.NewRouter(transporthttp.Dependencies{
		LoginExecutor:          loginService,
		CreateTicketExecutor:   createTicketService,
		UpdateTicketExecutor:   updateTicketService,
		GetTicketExecutor:      getTicketService,
		ListTicketsExecutor:    listTicketService,
		IngestDocumentExecutor: ingestDocumentService,
		ListDocumentsExecutor:  listDocumentsService,
		TokenVerifier:          tokenManager,
		RateLimiter:            redisRateLimiter,
		PublicRateLimit:        config.PublicRateLimit,
		AuthenticatedRateLimit: config.AuthenticatedRateLimit,
		RateLimitWindow:        config.RateLimitWindow,
        GraphQLHandler:         graphQLHandler,
	})

	// -------------------------
	// API Server
	// -------------------------
	server := &http.Server{
		Addr:              config.HTTPAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return APIRuntime{
		Server: server,
		Close: func() {
			closer.Close()
		},
	}, nil
}
