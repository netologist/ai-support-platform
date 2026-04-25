package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netologist/ai-support-platform/internal/app/command"
	infraauth "github.com/netologist/ai-support-platform/internal/infra/auth"
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
	// -------------------------
	// Auth
	// -------------------------
	tokenManager := infraauth.NewTokenManager(config.JWTIssuer, config.JWTSecret, config.JWTTTL)

	// -------------------------
	// Services
	// -------------------------
	loginService := command.NewLoginService(
		authRepository,
		infraauth.PasswordVerifier{},
		tokenManager,
	)

	// -------------------------
	// HTTP
	// -------------------------
	router := transporthttp.NewRouter(transporthttp.Dependencies{
		LoginExecutor: loginService,
		TokenVerifier: tokenManager,
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
