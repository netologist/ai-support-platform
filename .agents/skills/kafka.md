# Kafka Skill

## Use when

- Designing event flows, background workers, retry handling, or stream-based integrations.

## Conventions

- Define message keys, topic ownership, retry policy, and idempotency before implementation.
- Keep consumers safe to restart and replay.
- Prefer explicit dead-letter handling over silent drop behavior.
- Treat ordering guarantees as partition-scoped, not global.
- Keep producer and consumer contracts versioned and documented.
