package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/app/query"
	infraaudit "github.com/netologist/ai-support-platform/internal/infra/audit"
	infraauth "github.com/netologist/ai-support-platform/internal/infra/auth"
	infracache "github.com/netologist/ai-support-platform/internal/infra/cache"

	messagingkafka "github.com/netologist/ai-support-platform/internal/infra/messaging/kafka"
	postgresrepo "github.com/netologist/ai-support-platform/internal/infra/repository/postgres"
	"github.com/netologist/ai-support-platform/internal/infra/repository/sqlc"
	transporthttp "github.com/netologist/ai-support-platform/internal/transport/http"
	"github.com/redis/go-redis/v9"
)

type APIRuntime struct {
	Server  *http.Server
	CloseFn func()
}

func NewAPIRuntime(ctx context.Context, config Config) (APIRuntime, error) {
	closer := &closerStack{}

	// -------------------------
	// DB
	// -------------------------
	db, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		return APIRuntime{}, fmt.Errorf("connect postgres: %w", err)
	}
	closer.Add(db.Close)

	// -------------------------
	// Redis
	// -------------------------
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       config.RedisDatabase,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		closer.Close()
		return APIRuntime{}, fmt.Errorf("connect redis: %w", err)
	}
	closer.AddWithError(redisClient.Close)

	// -------------------------
	// Kafka
	// -------------------------
	kafkaPublisher, err := messagingkafka.NewPublisher(config.KafkaBrokers)
	if err != nil {
		closer.Close()
		return APIRuntime{}, fmt.Errorf("build kafka publisher: %w", err)
	}
	closer.Add(kafkaPublisher.Close)

	// -------------------------
	// Repositories
	// -------------------------
	queries := sqlc.New(db)
	authRepository := postgresrepo.NewAuthRepository(queries)
	auditRepository := postgresrepo.NewAuditRepository(queries)
	ticketRepository := postgresrepo.NewTicketRepository(queries)

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
		kafkaPublisher,
		config.KafkaTicketTopic,
	)

	updateTicketService := command.NewUpdateTicketService(
		ticketRepository,
		authorizer,
		ticketCache,
		auditLogger,
		kafkaPublisher,
		config.KafkaTicketTopic,
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

	// -------------------------
	// HTTP
	// -------------------------
	router := transporthttp.NewRouter(transporthttp.Dependencies{
		LoginExecutor:        loginService,
		CreateTicketExecutor: createTicketService,
		UpdateTicketExecutor: updateTicketService,
        GetTicketExecutor:    getTicketService,
        ListTicketsExecutor:  listTicketService,
		TokenVerifier:        tokenManager,
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
		CloseFn: func() {
			closer.Close()
		},
	}, nil
}
