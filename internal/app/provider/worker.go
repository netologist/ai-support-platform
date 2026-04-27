package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/netologist/ai-support-platform/internal/infra/cache/idempotency"
	messagingkafka "github.com/netologist/ai-support-platform/internal/infra/messaging/kafka"
	"github.com/netologist/ai-support-platform/internal/infra/messaging/outbox"
	"github.com/netologist/ai-support-platform/internal/infra/messaging/worker"
	"github.com/netologist/ai-support-platform/internal/infra/messaging/worker/consumers"
	postgresrepo "github.com/netologist/ai-support-platform/internal/infra/repository/postgres"
	generated "github.com/netologist/ai-support-platform/internal/infra/repository/sqlc"
)

type WorkerRuntime struct {
	Run   func(context.Context) error
	Close func()
}

func NewWorkerRuntime(ctx context.Context, config Config) (WorkerRuntime, error) {
	closer := &closerStack{}

	databasePool, err := buildDB(ctx, config, closer)
	if err != nil {
		closer.Close()
		return WorkerRuntime{}, err
	}

	redisClient, err := buildRedis(ctx, config, closer)
	if err != nil {
		closer.Close()
		return WorkerRuntime{}, err
	}

	kafkaPublisher, err := buildKafkaPublisher(config, closer)
	if err != nil {
		closer.Close()
		return WorkerRuntime{}, err
	}

	queries := generated.New(databasePool)
	outboxRepo := postgresrepo.NewOutboxRepository(queries)

	relay := outbox.NewRelay(outboxRepo, kafkaPublisher, config.KafkaTicketTopic, 100, 2*time.Second)

	idempotencyStore := idempotency.NewStore(redisClient, 24*time.Hour)
	ticketHandler := consumers.NewTicketEventHandler(idempotencyStore)

	consumer, err := messagingkafka.NewConsumer(config.KafkaBrokers, "worker-group", []string{config.KafkaTicketTopic})
	if err != nil {
		closer.Close()
		return WorkerRuntime{}, fmt.Errorf("build kafka consumer: %w", err)
	}
	closer.Add(consumer.Close)

	consumer.RegisterHandler(config.KafkaTicketTopic, ticketHandler.Handle)

	pool := worker.NewPool(config.WorkerPoolSize, config.WorkerPoolQueueSize)

	return WorkerRuntime{
		Run: func(runCtx context.Context) error {
			go relay.Run(runCtx)
			pool.Start(runCtx)
			defer pool.Stop()

			return consumer.Run(runCtx)
		},
		Close: func() {
			_ = closer.Close()
		},
	}, nil
}
