# Data And Messaging Rules

- PostgreSQL is the source of record; migrations must be forward-safe and repeatable.
- Redis should be treated as a cache or ephemeral coordination store, not a source of truth.
- Kafka consumers and producers must define ownership of message schema, retries, and idempotency.
- Avoid hidden background work; make queue, worker, and retry behavior explicit in code and config.
