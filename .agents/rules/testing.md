# Testing Rules

- Default to `-race` for local and CI test execution.
- Use `testify` for assertions when it improves readability.
- Use `goleak` in concurrency-heavy packages where goroutine lifecycle bugs are plausible.
- Use `testcontainers-go` only for integration cases that need real service behavior.
- Keep unit tests fast and deterministic; keep container-backed tests isolated and explicit.
