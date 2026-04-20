# Golang Skill

## Use when

- Implementing services, handlers, workers, packages, interfaces, or tests.

## Conventions

- Prefer small packages with explicit dependencies.
- Keep exported APIs minimal.
- Return wrapped errors with enough context to debug failures.
- Prefer `context.Context` on request-scoped and IO-bound operations.
- Favor composition over inheritance-style abstractions.

## Repo tools

- Format with `go tool task fmt`.
- Lint with `go tool task lint`.
- Test with `go tool task test`.
- Generate artifacts with `go tool task generate`.
