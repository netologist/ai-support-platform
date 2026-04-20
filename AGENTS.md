# AGENTS

This repository is a Go backend service workspace that uses Go-managed CLI tools and Task as the main command runner.

## Project defaults

- Prefer `go tool task` over raw shell commands when an equivalent task exists.
- Keep test commands race-enabled by default.
- Use `goimports` formatting and `golangci-lint` for shared linting.
- Use `compose.yaml` for local infrastructure dependencies.
- Keep generated code reproducible through explicit generator commands, not manual edits.

## Repo workflow

- Install project tools from `go.mod` before running local workflows.
- Use `.env.example` as the source for local environment variables.
- Use `go tool air` for hot reload and set `AIR_BUILD_TARGET` when switching binaries.
- Run database migrations through `go tool task migrate -- ...`.
- Run mock generation through `go tool task mocks -- ...`.

## Structure guidance

- Prefer `cmd/...` entrypoints for binaries such as `api` and `worker`.
- Keep transport, service, and persistence concerns separated.
- Treat PostgreSQL, Redis, Kafka, and HTTP APIs as boundary layers with thin adapters.
- Avoid leaking infrastructure details into domain logic.

## Quality bar

- Write table-driven tests where they improve coverage clarity.
- Add integration tests only where real infrastructure behavior matters.
- Keep functions small enough that control flow remains obvious without comments.
- Introduce abstractions only when they reduce coupling or improve testability.

See `.agents/rules` for project rules and `.agents/skills` for domain-specific implementation guidance.
