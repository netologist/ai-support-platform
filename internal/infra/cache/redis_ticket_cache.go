package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type RedisTicketCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisTicketCache(client *redis.Client, ttl time.Duration) RedisTicketCache {
	return RedisTicketCache{client: client, ttl: ttl}
}

func (cache RedisTicketCache) GetTicket(ctx context.Context, ticketID uuid.UUID) (entity.Ticket, bool, error) {
	payload, err := cache.client.Get(ctx, cache.key(ticketID)).Bytes()
	if err == redis.Nil {
		return entity.Ticket{}, false, nil
	}
	if err != nil {
		return entity.Ticket{}, false, err
	}

	var ticket entity.Ticket
	if err := json.Unmarshal(payload, &ticket); err != nil {
		// Corrupted cache entry — treat as a miss so the caller falls back to DB.
		slog.Warn("redis ticket cache: corrupted entry, treating as cache miss",
			slog.String("ticket_id", ticketID.String()),
			slog.Any("error", err),
		)
		return entity.Ticket{}, false, nil
	}

	return ticket, true, nil
}

func (cache RedisTicketCache) SetTicket(ctx context.Context, ticket entity.Ticket) error {
	payload, err := json.Marshal(ticket)
	if err != nil {
		return err
	}

	return cache.client.Set(ctx, cache.key(ticket.ID), payload, cache.ttl).Err()
}

func (cache RedisTicketCache) DeleteTicket(ctx context.Context, ticketID uuid.UUID) error {
	return cache.client.Del(ctx, cache.key(ticketID)).Err()
}

func (cache RedisTicketCache) key(ticketID uuid.UUID) string {
	return fmt.Sprintf("ticket:%s", ticketID)
}
