# Project Rules

- Use `go tool task` commands as the default local workflow entrypoint.
- Keep configuration in committed files when it is shared, and in `.env` for local secrets.
- Prefer additive changes to project tooling instead of replacing the existing task-based setup.
- Keep repository files ASCII unless a file already uses Unicode.
- Do not add dependencies without a clear runtime, test, or tooling reason.
