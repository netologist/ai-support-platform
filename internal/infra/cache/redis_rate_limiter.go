package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type RedisRateLimiter struct {
	client *redis.Client
}

func NewRedisRateLimiter(client *redis.Client) RedisRateLimiter {
	return RedisRateLimiter{client: client}
}

func (limiter RedisRateLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (service.RateLimitResult, error) {
	cacheKey := fmt.Sprintf("ratelimit:%s", key)
	count, err := limiter.client.Incr(ctx, cacheKey).Result()
	if err != nil {
		return service.RateLimitResult{}, err
	}

	if count == 1 {
		if err := limiter.client.Expire(ctx, cacheKey, window).Err(); err != nil {
			return service.RateLimitResult{}, err
		}
	}

	ttl, err := limiter.client.TTL(ctx, cacheKey).Result()
	if err != nil {
		return service.RateLimitResult{}, err
	}

	return service.RateLimitResult{
		Allowed:    count <= limit,
		Count:      count,
		Limit:      limit,
		RetryAfter: ttl,
	}, nil
}
