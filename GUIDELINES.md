# Project Guidelines

These instructions apply to the whole repository.

## Project Overview

`dwoe` is a Go CLI for running autonomous coding agents in isolated Docker containers. 

The main binary is built from `cmd/dwoe`. 

The migration helper is built from `cmd/migrate`. (Only exists to migrate old backups and deleted eventually.)

The CLI starts and manages worker containers, stores workspace data under the configured data directory, and includes a web dashboard for monitoring workers, logs, batches, and diffs.

## Repository Layout

- `cmd/`: CLI entry points. Keep `main.go` thin and delegate behavior to packages.
- `internal/`: application packages for CLI, Docker, workspace, state, config, logging, migrations, and utilities.
- `web/`: web dashboard handlers, templates, static assets, and tests.
- `schema/`: schema-related code and tests.
- `docker/`: Dockerfiles for default agent and proxy images.
- `examples/`: walkthrough projects and task YAML examples.
- `test/integration/`: integration tests, Docker-backed fixtures, and integration testdata.

## Development Commands

- `make build`: build `bin/dwoe` from `./cmd/dwoe`.
- `make test`: run `go test ./...`.
- `make test-v`: run verbose unit tests.
- `make lint`: run formatting, vet, and `golangci-lint run`.
- `make all`: run lint, tests, and build.
- `make dev`: run the Air hot-reload dashboard command configured in `.air.toml`.
- `make integration-test`: run integration tests with `-tags=integration`; Docker is required.

Before considering code complete, run the narrowest relevant tests first, then prefer `make test` and `make lint` when practical.

## Runtime Requirements

- Go 1.25.x
- Docker for normal `dwoe` operation and integration tests
- `golangci-lint` for `make lint`
- `air` for `make dev`

## Agent Commands

Reusable agent command prompts can live under `agents/commands/`. Keep these prompts agent-agnostic so different tools can read and reuse them to shape behavior. This directory is ignored by Git because it is local workflow configuration, not project source.

## Development Mode: Interactive & Incremental

When running in the `/iterator` mode, this is a guided, step-by-step development session.

1. One small step at a time: wait for approval before proceeding
2. Ask before creating: describe what you plan to write, get confirmation
3. Small files, small changes: prefer 20-50 lines per step
4. Longer runs allowed if enabled: when the user explicitly permits, proceed but stay within the current task scope only
5. Minimal code: only the necessary logic, no excessive defensive programming
6. No docstrings until requested: documentation is added as the final step after implementation is complete

## Workflow

1. Read the plan/TODO
2. Describe the first small piece to implement
3. Wait for approval until running in "don't ask" mode
4. Implement just that piece
5. Verify it compiles
6. Wait for the next instruction

## Conventions

- Follow standard Go conventions: https://go.dev/doc/effective_go
- Use `gofmt`/`goimports` formatting. Do not manually format
- Prefer the standard library
- Only introduce third-party packages when the standard library lacks the necessary functionality
- Use `go mod tidy` to keep `go.mod` clean after dependency changes
- Export only what needs to be exported. Keep the public API surface small
- Return errors rather than panicking
- Never discard errors with `_` unless there is a clear reason
- Wrap errors with `fmt.Errorf("context: %w", err)` to preserve the error chain
- Use guard clauses and early returns over deeply nested if/else
- Keep comments sparse and useful; avoid adding doc comments unless documentation is part of the task

## Restrictions

- DO NOT create entire packages in one go unless explicitly permitted
- DO NOT add extra guards, checks, or error handling beyond what is necessary
- DO NOT write docstrings or comments during implementation unless requested
- DO NOT expand the scope beyond the current task
- DO NOT rewrite unrelated code, generated assets, examples, or release artifacts
- Preserve existing user changes in the working tree

## Testing

- Add focused tests near the package being changed
- Use table-driven tests for functions with multiple input/output cases using `[]struct{...}` and `for _, tt := range tests { ... }`
- Use `t.Helper()` in test helper functions
- Use `t.TempDir()` for temp directories
- Use `TestMain` for global fixtures and only when strictly necessary
- Use `testdata/` directories for test fixtures
- Name test cases descriptively: `TestFunctionName_condition_expectedResult`
- Integration tests may require Docker images and should be run with `make integration-test`

## Project Structure

- Follow the standard Go project layout: `cmd/`, `internal/`, `pkg/` where applicable
- Keep `main.go` thin: delegate to packages under `internal/`
- One package per directory. The package name matches the directory name

## Linting

The project uses `.golangci.yml` with `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `gocritic`, and `revive`. Fix lint findings rather than suppressing them unless there is a specific reason.

## Verification

Before considering work complete, run the relevant subset of:

```bash
go vet ./...
golangci-lint run ./...
go test ./...
```

For the standard project flow, these are also available through:

```bash
make test
make lint
make build
```

## Version Control

- Do not create commits unless the user asks
- If requested to commit multiple changes or new files, organize them into separate chunks of related work
- If requested to make a specific commit like `git commit -m "feat: new feature"`, use the message verbatim, no extra content
- DO NOT add any self-references, agent attribution, or co-authored tags into commit messages
- DO NOT change a commit message or use bullet-point lists until requested
- All commits are authored by the currently configured git user
- Sign commits with `git commit -S`; SSH signing is configured for this repository
- If signing fails because the key is unavailable, prompt the user to add the signing key to the SSH agent
