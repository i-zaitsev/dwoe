# Changelog

## v0.1.2 (2026-11-10)

### Features

- Web dashboard rebuilt on Go templates and htmx (#8)
- JSON API under `/api/v1`
- Standalone `cmd/web` and `dwoe web --ui-dir`
- Added `dwoe web --host` and `--port` next to `--addr`
- Highlighted JSON in logs, foldable sections, link to the result
- Multi-provider agents: `anthropic` and `openai`
- The `--provider` flag in `fire`
- Task description in `inspect -desc` and the dashboard
- Options API for task configs
- Layered and universal Docker images, `make images`

### Fixes

- The `--model` override applies to the task
- No panic on unknown provider names
- Provider credentials and domains from the registry
- Codex auth with `CODEX_API_KEY` or `OPENAI_API_KEY`
- Running workspaces survive cancelled dashboard requests
- Dashboard errors on polls, streams, and page loads
- Prompt files limited to the work dir, symlinks included
- Theme toggle keeps the current page

### Breaking

- The `dwoe-codex:latest` image removed: use `dwoe-agent:*` and `provider: openai`
- Agent, git, and resources are optional task sections
- Sentinel hashes changed: completed workspaces re-run once

### Docs

- Providers, authentication, and the image tree
- Dashboard, project layout, and lint rules

### Other

- Dashboard in `pkg/hio`, schema in `internal/`, thin entry points
- CLI framework and assertions from `github.com/i-zaitsev/pkg`
- Text log format via `--logfmt=text`
- Colors only on a terminal
- Default models `claude-sonnet-5` and `gpt-5.6-terra`
- Base image on Debian Trixie
- Consistent command output wording
- Lint enforces dashboard layering, pins the Go toolchain
- CI type-checks the integration build tag

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
