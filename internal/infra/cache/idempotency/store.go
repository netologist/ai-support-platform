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

// CheckAndMark atomically checks if key was already processed, and marks it
// as processed in a single Redis SET NX command. Returns true if the key was
// newly set (i.e. not yet processed), false if it was already present.
// Two separate Exists + Set calls would have a TOCTOU race under concurrent
// consumers — the atomic SET NX eliminates that window.
func (s *Store) CheckAndMark(ctx context.Context, key string) (bool, error) {
	ok, err := s.client.SetNX(ctx, s.prefix+key, "1", s.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("idempotency check-and-mark: %w", err)
	}

	// ok == true  → key was newly set → NOT yet processed
	// ok == false → key already existed → already processed
	return ok, nil
}

// IsProcessed checks whether the key has been processed before.
// Prefer CheckAndMark for actual deduplication to avoid TOCTOU races.
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
