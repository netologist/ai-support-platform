# PostgreSQL Skill

## Use when

- Designing schemas, writing migrations, adding queries, or reviewing persistence behavior.

## Conventions

- Prefer additive migrations and reversible rollbacks where practical.
- Keep schema changes backward-compatible during rolling deploys.
- Index based on actual read paths, not guesswork.
- Use transactions for multi-step state changes that must stay consistent.
- Keep SQL close to the persistence layer and avoid leaking raw SQL across the codebase.

## Repo tools

- Run migration commands through `go tool task migrate -- ...`.
- Keep `DATABASE_URL` aligned with `.env` and `compose.yaml`.
