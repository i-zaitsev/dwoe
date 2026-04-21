# Changelog

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
