# Default Agent Images

Example Dockerfiles for `dwoe` agents to work with Python, Go, C, and C++ codebases using Claude Code agent.

Proxy image gives basic web isolation for workers running in containers. (Required unless `--noproxy` is used.)

## Build

```bash
# Universal image (default)
docker build -f docker/Dockerfile.base -t dwoe-agent:latest docker/

# Go-only
docker build -f docker/Dockerfile.claude-go -t dwoe-agent:go docker/

# Python-only
docker build -f docker/Dockerfile.claude-python -t dwoe-agent:python docker/

# C-only (C23, gcc)
docker build -f docker/Dockerfile.claude-c -t dwoe-agent:c docker/

# C++-only (C++20, g++)
docker build -f docker/Dockerfile.claude-cpp -t dwoe-agent:cpp docker/

# Proxy (required unless --noproxy is used)
docker build -f docker/Dockerfile.proxy -t dwoe-proxy:latest docker/
```

## Alternative Images 

There is no requirement to use provided images. Set `agent.image` option in YAML file to use any Docker image that 
runs agentic loop. The only requirements are:

* The image must have an entrypoint that runs the agent
* `/workspace` is mounted with the source code (read-write)
* `/logs` is mounted for log output (read-write)


## Environment Variables

The following env vars are set by `dwoe` when starting a container:

| Variable                     | Description                             |
|------------------------------|-----------------------------------------|
| `WORKSPACE_ID`               | Unique workspace identifier             |
| `WORKSPACE_NAME`             | Human-readable workspace name           |
| `CLAUDE_MODEL`               | Model to use                            |
| `MAX_TURNS`                  | Maximum agent turns                     |
| `GIT_USER_NAME`              | Git commit author name                  |
| `GIT_USER_EMAIL`             | Git commit author email                 |
| `TASK_PROMPT`                | Initial prompt for the agent (optional) |
| `HTTP_PROXY` / `HTTPS_PROXY` | Proxy URL (when proxy is enabled)       |

## Entrypoint

The bundled `entrypoint.sh` sets up git, runs Claude Code in a retry
loop, and uses `TASK_PROMPT` to tell the agent what to do. It reads
all configuration from environment variables.

Custom images can ignore this entrypoint entirely and define their own.
