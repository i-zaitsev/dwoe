# dwoe: dockerized autonomous coding agents

A simple CLI tool for running autonomous agents in isolated Docker containers.

> This is the alpha version software developed for private use! Avoid using it for critical tasks or production setup! 

## Requirements

- Go 1.25+
- Docker

## Quick Start

1. Start Docker daemon
2. Build example images (see [/docker](docker) folder) or bring a custom one
3. Export `CLAUDE_CODE_OAUTH_TOKEN` from `claude setup-token` or set up `OPENAI_API_KEY`
4. Build the binary and run
```bash
make build
./dwoe fire --repo=/path/to/src --do="create a new package to format time"
```

The `fire` command launches a container, mounts the input folder, and runs an agentic loop to implement the requirements
in the prompt. There is the `--work` flag to define a prompt file. See other sections for more commands. Check out 
the [examples](examples) folder to see more use-cases.

## Usage

There are multiple commands to run and inspect workers.
```
dwoe <version>

Usage:
	dwoe [flags] <command> [args]

Flags:
	--datadir <dir>     Data directory (default: ~/.dwoe)
	--logfile <path>    Write JSON logs to file
	--loglevel <level>  Log level: debug, info, warn, error (default: warn)
	--logfmt <format>   Log format: text (human), json (default)
	--noproxy           Disable proxy container

Commands:
  batch    <dir>                      Run all task files in parallel
  collect  <name|id> [--batch ID]     Collect commits into a repo branch
  create   <task.yaml>                Create workspace from config
  destroy  <name|id>                  Remove workspace
  fire     --repo <url|path> [flags]  Quick-start workspace from repo
  inspect  <name|id>                  Show detailed workspace info
  list     [--format FMT]             List workspaces
  logs     <name|id>                  Show workspace logs
  patches  <name|id> --dir <dir>      Export patches to directory
  run      <task.yaml>                Create and start workspace
  start    <name|id>                  Start existing workspace
  status   <name|id>                  Show workspace status
  stop     <name|id>                  Stop running workspace
  version                             Show version
  web      [--addr HOST:PORT | --host HOST --port PORT] [--ui-dir DIR] Start web dashboard
```
The default location for workspace data, including logs, code artifacts, and task config, is `~/.dwoe`.

Use `dwoe list` to show the list of running and finished workers.

Use `dwoe inspect <name|id>` to get information about the worker.

Each `dwoe fire|run` starts two containers: 
* The one with agentic loop implementing the task, and
* The proxy container that prevents arbitrary web requests.

There are a few [Docker files](docker) to build default agent containers. A custom container can be used instead.
See examples and [default config](/internal/config/defaults.go). Use YAML files and `run` command to redefine the 
image, or build the example images with `make images`.

Every agent image carries both agent CLIs, so the same image runs any provider:
`dwoe-agent:latest` is the universal default, with smaller `dwoe-agent:go|python|c|cpp`
variants per language. See [docker/README.md](docker/README.md).

## Providers

The agent backend is selected with `agent.provider` in the task file, or `--provider` on
`dwoe fire`. Supported values are `anthropic` (default) and `openai`.

## Authentication

Credentials are passed through from the host environment; nothing is stored by `dwoe`.

| Provider    | Environment variable                             |
|-------------|--------------------------------------------------|
| `anthropic` | `CLAUDE_CODE_OAUTH_TOKEN` or `ANTHROPIC_API_KEY` |
| `openai`    | `CODEX_API_KEY` or `OPENAI_API_KEY`              |

Build a custom image to modify the entry point and provide the authentication credentials differently.

## Task file

Running a worker from a task file:

```bash
dwoe run task.yaml
```

The file format:
```yaml
name: my-task
source:
  local_path: ./repo
  prompt_file: ./prompt.md
agent:
  provider: anthropic
  image: dwoe-agent:go
  model: claude-sonnet-5
  max_turns: 10
  env_vars:
    CLAUDE_CODE_OAUTH_TOKEN: ${CLAUDE_CODE_OAUTH_TOKEN}
git:
  user_name: "dwoe-agent"
  user_email: "agent@dwoe.dev"
resources:
  cpu: "2"
  memory: "4G"
```

## Web Dashboard

Use `dwoe web` to start a dashboard for monitoring workers, streaming logs, and viewing workspace status in the browser.

```bash
dwoe web --port 9090
dwoe web --host 0.0.0.0
dwoe web --addr 0.0.0.0:9090
```

The dashboard listens on 127.0.0.1:8080 by default. `--host` and `--port` change one part,
`--addr` sets both and cannot be combined with them.

The dashboard is server-rendered HTML with htmx, served together with a JSON API under `/api/v1`.
Templates and static files are embedded in the binary; `--ui-dir pkg/hio/ui` reads them from disk instead.

## Examples

See [`examples/`](examples) for complete walkthroughs covering quick start, single tasks, Python projects, batch parallel runs, patch collection, batch-and-merge workflows, and custom prompt configuration.

## Development

```bash
make test
make lint
make all
```

To work on the dashboard, run `make dev`. It starts `cmd/web` with Air hot reload and debug logging on 127.0.0.1:8080.
Go changes trigger a rebuild; template and CSS changes under `pkg/hio/ui` show up on the next request.