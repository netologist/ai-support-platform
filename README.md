# AI Support Platform

This repository uses Go's tool management together with Task to provide a single entrypoint for development commands.

## Prerequisites

- Go 1.26+
- VS Code with the Go extension, or GoLand

## Install tools

Install the project-managed CLI tools defined in `go.mod`:

```bash
go install tool
```

If your Go version does not support that workflow yet, you can still run commands through Go directly with `go tool ...` after dependencies are downloaded.

This includes Delve for debugging because `github.com/go-delve/delve/cmd/dlv` is tracked in the Go tool block.

## Editor setup

If you use VS Code, install the official Go extension: `golang.go`.

The workspace settings in `.vscode/settings.json` are configured to:

- enable format on save
- use `goimports` as the formatter
- enable in-editor test discovery
- run lint on save with `golangci-lint`
- pass `-race` to test runs in the editor

Debugging is configured in `.vscode/launch.json` for:

- launching the current package
- launching the current file
- attaching to a Delve server on port `2345`

To install Delve locally for the project toolchain:

```bash
go install tool
```

To install only Delve directly:

```bash
go tool dlv version
```

If the binary is not installed yet, run:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

## Task commands

List available tasks:

```bash
go tool task --list
```

Install repo-managed tools and verify key editor tooling:

```bash
go tool task bootstrap
```

Run the development server with hot reload:

```bash
go tool task run
```

Start Delve for the API entrypoint when `cmd/api` exists:

```bash
go tool task debug:api
```

Start Delve for the worker entrypoint when `cmd/worker` exists:

```bash
go tool task debug:worker
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

Run containerized black-box E2E tests:

```bash
go tool task e2e:up
go tool task e2e:test
go tool task e2e:down
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
- `debug:api` and `debug:worker` require `cmd/api` and `cmd/worker` directories to exist.
- The E2E stack is defined in `compose.e2e.yaml` and exposes the API at `http://localhost:18080`.
- E2E scenarios are implemented with Ginkgo/Gomega in `test/e2e`.
