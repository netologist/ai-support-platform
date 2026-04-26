package idempotency

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

func NewStore(client *redis.Client, ttl time.Duration) *Store {
	return &Store{client: client, ttl: ttl, prefix: "idempotency:"}
}

func (s *Store) IsProcessed(ctx context.Context, key string) (bool, error) {
	result, err := s.client.Exists(ctx, s.prefix+key).Result()
	if err != nil {
		return false, fmt.Errorf("idempotency check: %w", err)
	}

	return result > 0, nil
}

func (s *Store) MarkProcessed(ctx context.Context, key string) error {
	return s.client.Set(ctx, s.prefix+key, "1", s.ttl).Err()
}
