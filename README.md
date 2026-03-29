# dwoe: dockerized autonomous coding agents

A simple CLI tool for running autonomous Claude Code agents in isolated Docker containers.

> This is alpha version software developed for private use!
> Avoid using it for critical tasks or production setup. 

## Requirements

- Go 1.25+
- Docker

## Quick Start

1. Start Docker daemon
2. Build example images (see [/docker](docker) folder)
3. Export `CLAUDE_CODE_OAUTH_TOKEN` from `claude setup-token`
4. Build the binary and run
```bash
make build
./dwoe fire --repo=/path/to/src --do="create a new package to format time"
```
The `fire` command launches a container, mounts the input folder, and runs an agentic loop to implement the requirements
in the prompt. There is a `--work` flag to define a prompt file. See other sections for more commands. Check out 
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
	--logfmt <format>   Log format: text, json (default: json)
	--no-proxy          Disable proxy container

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
  web      [--addr ADDR]              Start web dashboard
```
The default location for workspace data, including logs, code artifacts, and task config, is `~/.dwoe`.

Use `dwoe list` to show the list of running and finished workers.

Use `dwoe inspect <name|id>` to get information about the worker.

Each `dwoe fire|run` starts two containers: 
* The one with agentic loop implementing the task, and
* The proxy container that prevents arbitrary web requests.

There are a few [Docker files](docker) to build default agent containers. A custom container can be used instead.
See examples and [default config](/internal/config/defaults.go). Use YAML files and `run` command to redefine the 
image, or build example images with `dwoe-agent:latest` and `dwoe-proxy:latest` names.

## Authentication

Each spawned worker looks for `CLAUDE_CODE_OAUTH_TOKEN` to authenticate `claude` binary requests. 

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
  image: dwoe-agent:go
  model: claude-sonnet-4-6
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
dwoe web --addr :9090
```

## Examples

See [`examples/`](examples) for complete walkthroughs covering quick start, single tasks, Python projects, batch parallel runs, patch collection, batch-and-merge workflows, and custom prompt configuration.

## Development

```bash
make test
make lint
make all
```

Note that the project includes `.air.toml` and `make dev` allows hot reloading upon changes. 