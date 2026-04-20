# Redis Skill

## Use when

- Adding caching, distributed locks, ephemeral state, rate limits, or short-lived queues.

## Conventions

- Set TTLs intentionally.
- Design keys with a stable namespace pattern.
- Treat cache misses as normal behavior.
- Avoid storing canonical business state only in Redis.
- Define invalidation behavior before introducing a cache.
