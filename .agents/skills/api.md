# API Skill

## Use when

- Adding HTTP endpoints, GraphQL operations, request validation, or API-facing error handling.

## Conventions

- Keep transport concerns thin and move business logic into services.
- Validate inputs at the boundary.
- Use stable, documented error shapes.
- Make timeouts, retries, and authentication behavior explicit.
- Keep handlers observable with structured logs and request-scoped context.
