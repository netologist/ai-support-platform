package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/netologist/ai-support-platform/internal/domain/service"
)

// rateLimitScript atomically increments a counter and sets its TTL on first use.
// Returns [count, ttl_seconds]. The TTL is only set when count == 1 so that
// existing windows are never reset by a race between INCR and EXPIRE.
var rateLimitScript = redis.NewScript(`
local key    = KEYS[1]
local window = tonumber(ARGV[1])
local count  = redis.call("INCR", key)
if count == 1 then
	redis.call("EXPIRE", key, window)
end
local ttl = redis.call("TTL", key)
return {count, ttl}
`)

type RedisRateLimiter struct {
	client *redis.Client
}

func NewRedisRateLimiter(client *redis.Client) RedisRateLimiter {
	return RedisRateLimiter{client: client}
}

func (limiter RedisRateLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (service.RateLimitResult, error) {
	cacheKey := fmt.Sprintf("ratelimit:%s", key)
	windowSecs := int64(window.Seconds())

	res, err := rateLimitScript.Run(ctx, limiter.client, []string{cacheKey}, windowSecs).Int64Slice()
	if err != nil {
		return service.RateLimitResult{}, err
	}

	count := res[0]
	ttl := time.Duration(res[1]) * time.Second

	return service.RateLimitResult{
		Allowed:    count <= limit,
		Count:      count,
		Limit:      limit,
		RetryAfter: ttl,
	}, nil
}
