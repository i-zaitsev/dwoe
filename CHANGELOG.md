# Changelog

## Unreleased

### Features

- Multi-provider agents basic support: `anthropic` and `openai`
- The `fire` command supports `--provider` flag
- Task description shown by `dwoe inspect -desc` and in the web dashboard
- Task configs are built through an options API
- Layered Docker images for agents
- Universal image with Go, Python, C, and C++ toolchains
- The `make images` target to build the whole image tree

### Fixes

- The `--model` override had no effect and now applies to the task
- Unknown provider names no longer panic
- Provider credentials and allowed domains are resolved from the provider registry
- `codex exec` authenticates with `CODEX_API_KEY` and falls back to `OPENAI_API_KEY`

### Breaking

- `dwoe-codex:latest` is gone: use any `dwoe-agent:*` image with `provider: openai`
- Agent, git, and resources are optional task sections filled from globals and defaults
- Sentinel hashes changed: a workspace completed by an earlier release re-runs once

### Other

- Human-readable log format via `--logfmt=text`, with the source location on warnings and errors
- Colors only when the log writer is a terminal
- Default models are now `claude-sonnet-5` and `gpt-5.6-terra`
- Base image moved to Debian Trixie
- Consistent `Workspace created|started` wording in command output
- CI type-checks the integration build tag

### Docs

- Provider key, authentication per provider, and the image tree

## v0.1.1 (2026-04-21)

### Features

- A pretty version of the logs’ page (#2)
- Task view duration and config
- Added global `--taskname` flag
- Added `c` and `cpp` docker images
- Explicit `err` for completed workspaces
- Sentinel writing on task completion
- Relative source paths in log output
- Resuming policy in run and create
- Recursive task discovery and relative branch names
- Overrides in `run` command
- Global task flags

### Fixes

- Prompt file copied to the agent container
- Marshal round-trip for `Continue` policy
- A message format updated for `assert.Condition`
- Dedup names with suffix if provided
- Batch command applies `sourcedir` fallback

### Docs

- Resume policy in examples

### Other

- Consistent flag naming
- Test setup refactored
- Test helpers extended
- Explicit `defer func() { _ = closer.Close() }` pattern

## v0.1.0 (2026-03-29)

Initial public release.

### Features

- `dwoe fire` command for quick-start workspaces from repo and prompt
- `dwoe run` command with full `task.yaml` configuration
- `dwoe batch` for parallel task execution
- `dwoe collect|patches` for gathering agent work
- `dwoe web` to run a web dashboard with live log streaming
- Network isolation via Squid proxy `allowlist.txt`
- Docker-based workspace lifecycle (create, start, stop, destroy)
