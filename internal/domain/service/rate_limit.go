package service

import (
	"context"
	"time"
)

type RateLimitResult struct {
	Allowed    bool
	Count      int64
	Limit      int64
	RetryAfter time.Duration
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int64, window time.Duration) (RateLimitResult, error)
}
