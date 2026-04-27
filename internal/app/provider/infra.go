package provider

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	infraai "github.com/netologist/ai-support-platform/internal/infra/ai"
	messagingkafka "github.com/netologist/ai-support-platform/internal/infra/messaging/kafka"
	"github.com/redis/go-redis/v9"
)

func buildDB(ctx context.Context, config Config, closer *closerStack) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	closer.Add(db.Close)
	return db, nil
}

func buildRedis(ctx context.Context, config Config, closer *closerStack) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       config.RedisDatabase,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	closer.AddWithError(client.Close)
	return client, nil
}

func buildKafkaPublisher(config Config, closer *closerStack) (messagingkafka.Publisher, error) {
	publisher, err := messagingkafka.NewPublisher(config.KafkaBrokers)
	if err != nil {
		return messagingkafka.Publisher{}, fmt.Errorf("build kafka publisher: %w", err)
	}
	closer.Add(publisher.Close)
	return publisher, nil
}

func buildAI(ctx context.Context, config Config, closer *closerStack) (infraai.Providers, error) {
	providers, err := infraai.NewProviders(ctx, infraai.Config{
		Provider:       config.AIProvider,
		Model:          config.AIModel,
		EmbeddingModel: config.AIEmbeddingModel,
		APIKey:         config.AIAPIKey,
	})
	if err != nil {
		return infraai.Providers{}, fmt.Errorf("build AI provider: %w", err)
	}
	closer.AddWithError(providers.Close)
	return providers, nil
}
