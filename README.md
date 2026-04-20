# AI Support Platform

This repository uses Go's tool management together with Task to provide a single entrypoint for development commands.

## Prerequisites

- Go 1.26+

## Install tools

Install the project-managed CLI tools defined in `go.mod`:

```bash
go install tool
```

If your Go version does not support that workflow yet, you can still run commands through Go directly with `go tool ...` after dependencies are downloaded.

## Task commands

List available tasks:

```bash
go tool task --list
```

Run the development server with hot reload:

```bash
go tool task run
```

Format the codebase:

```bash
go tool task fmt
```

Run lint checks:

```bash
go tool task lint
```

Run tests with the race detector:

```bash
go tool task test
```

Run tests with coverage and write a JUnit report to `tmp/unit-tests.xml`:

```bash
go tool task coverage
```

Generate an HTML coverage report at `tmp/coverage.html`:

```bash
go tool task coverage:html
```

Run vulnerability checks:

```bash
go tool task vuln
```

Run code generators:

```bash
go tool task generate
```

Pass goose arguments through Task:

```bash
go tool task migrate -- -dir db/migrations postgres "$DATABASE_URL" status
```

Pass mockery arguments through Task:

```bash
go tool task mocks -- --all --output internal/mocks
```

## Notes

- `test`, `coverage`, and `coverage:html` run with `-race` enabled.
- Coverage artifacts are written under `tmp/`.
- `migrate` and `mocks` are thin wrappers around `go tool goose` and `go tool mockery` so you can keep command usage flexible.