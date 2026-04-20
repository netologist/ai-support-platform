# GitHub Copilot Instructions

Use [AGENTS.md](../AGENTS.md) as the repository-level source of truth.

Follow the rules in these files:

- [../AGENTS.md](../AGENTS.md)
- [../.agents/rules/project.md](../.agents/rules/project.md)
- [../.agents/rules/architecture.md](../.agents/rules/architecture.md)
- [../.agents/rules/testing.md](../.agents/rules/testing.md)
- [../.agents/rules/data-and-messaging.md](../.agents/rules/data-and-messaging.md)

Use the domain skills in these files when relevant:

- [../.agents/skills/golang.md](../.agents/skills/golang.md)
- [../.agents/skills/postgresql.md](../.agents/skills/postgresql.md)
- [../.agents/skills/redis.md](../.agents/skills/redis.md)
- [../.agents/skills/kafka.md](../.agents/skills/kafka.md)
- [../.agents/skills/api.md](../.agents/skills/api.md)
- [../.agents/skills/clean-code.md](../.agents/skills/clean-code.md)

Practical defaults for this repository:

- Prefer `go tool task` over ad hoc shell commands when a task exists.
- Keep tests race-enabled by default.
- Use `goimports` for formatting.
- Use `compose.yaml` for local dependencies.
- Treat AGENTS and `.agents` files as authoritative; do not create conflicting guidance in new instruction files.
