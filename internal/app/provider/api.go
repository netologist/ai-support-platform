package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	messagingkafka "github.com/netologist/ai-support-platform/internal/infra/messaging/kafka"
	"github.com/redis/go-redis/v9"
)

type Runtime struct {
	Server  *http.Server
	CloseFn func()
}

type closerStack struct {
	fns []func() error
}

func (c *closerStack) Add(fn func()) {
	c.fns = append(c.fns, func() error { fn(); return nil })
}

func (c *closerStack) AddWithError(fn func() error) {
	c.fns = append(c.fns, fn)
}

func (c *closerStack) Close() error {
	var errs []error

	for i := len(c.fns) - 1; i >= 0; i-- {
		if err := c.fns[i](); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func NewRuntime(ctx context.Context, config Config) (Runtime, error) {
	closers := &closerStack{}

	// -------------------------
	// DB
	// -------------------------
	db, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		return Runtime{}, fmt.Errorf("connect postgres: %w", err)
	}
	closers.Add(db.Close)

	// -------------------------
	// Redis
	// -------------------------
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       config.RedisDatabase,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		closers.Close()
		return Runtime{}, fmt.Errorf("connect redis: %w", err)
	}
	closers.AddWithError(redisClient.Close)

	// -------------------------
	// Kafka
	// -------------------------
	kafkaPublisher, err := messagingkafka.NewPublisher(config.KafkaBrokers)
	if err != nil {
		closers.Close()
		return Runtime{}, fmt.Errorf("build kafka publisher: %w", err)
	}
	closers.Add(kafkaPublisher.Close)


    // -------------------------
    // API Server
    // -------------------------
	server := &http.Server{
		Addr:              config.HTTPAddress,
		Handler:           http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return Runtime{
		Server: server,
		CloseFn: func() {
			closers.Close()
		},
	}, nil
}
