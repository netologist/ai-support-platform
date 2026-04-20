# Architecture Rules

- Keep domain logic independent from HTTP, SQL, Redis, Kafka, and generated code.
- Use clear boundaries between transport, application/service, and infrastructure layers.
- Prefer constructor-based dependency injection over package globals.
- Keep side effects at the edges and pass interfaces inward only when they reduce coupling.
- Generated code belongs behind adapters rather than becoming the application core.
