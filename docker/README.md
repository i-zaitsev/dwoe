# Default Agent Images

Example Dockerfiles for `dwoe` agents to work with Python, Go, C, and C++ codebases.

Every agent image is built from `dwoe-base`, which carries binaries for agent CLIs
(Claude Code and OpenAI Codex). Images are therefore provider-agnostic: the same image
runs either provider, selected by `agent.provider` in the task file. Language images only
add a toolchain on top.

Proxy image gives basic web isolation for workers running in containers. A proxy container is 
required unless `--noproxy` is used when a task is started.

## Images

| Tag                 | Dockerfile             | Contents                           |
|---------------------|------------------------|------------------------------------|
| `dwoe-base:latest`  | `Dockerfile.base`      | Debian Trixie, git, agent CLIs     |
| `dwoe-agent:c`      | `Dockerfile.c`         | Base plus gcc (C23), cmake, gdb    |
| `dwoe-agent:cpp`    | `Dockerfile.cpp`       | Base plus g++ (C++20), cmake, gdb  |
| `dwoe-agent:go`     | `Dockerfile.golang`    | Base plus the Go toolchain         |
| `dwoe-agent:python` | `Dockerfile.python`    | Base plus Python and uv            |
| `dwoe-agent:latest` | `Dockerfile.universal` | Base plus Go, Python/uv, C and C++ |
| `dwoe-proxy:latest` | `Dockerfile.proxy`     | Alpine and Squid                   |

The `dwoe-agent:latest` is the default when a task does not set `agent.image`. Prefer a
language image when a smaller container is enough: the universal image carries every
toolchain and is larger.

## Build

```bash
# Everything: base is built first, with others targets depending on base
make images

# One at a time
make image-base
make image-universal   # dwoe-agent:latest
make image-go
make image-python
make image-c
make image-cpp
make image-proxy       # required unless --noproxy is used
```

## Alternative Images 

There is no requirement to use provided images. Set `agent.image` option in YAML file to use any Docker image that 
runs agentic loop. The only requirements are:

* The image must have an entrypoint that runs the agent
* `/workspace` is mounted with the source code (read-write)
* `/logs` is mounted for log output (read-write)

## Environment Variables

The following env vars are set by `dwoe` when starting a container:

| Variable                     | Usage                                               |
|------------------------------|-----------------------------------------------------|
| `WORKSPACE_ID`               | Sets a unique workspace identifier                  |
| `WORKSPACE_NAME`             | Sets a human-readable workspace name                |
| `AGENT_PROVIDER`             | Selected provider                                   |
| `AGENT_MODEL`                | Model to use (provider-agnostic)                    |
| `CLAUDE_MODEL`               | Model to use for Claude (same value as AGENT_MODEL) |
| `MAX_TURNS`                  | Maximum agent turns (Claude)                        |
| `GIT_USER_NAME`              | Git commit author name                              |
| `GIT_USER_EMAIL`             | Git commit author email                             |
| `TASK_PROMPT`                | Initial prompt for the agent (optional)             |
| `HTTP_PROXY` / `HTTPS_PROXY` | Proxy URL (when proxy is enabled)                   |
| `CLAUDE_CODE_OAUTH_TOKEN`    | API key for Claude Code                             |
| `CODEX_API_KEY`              | API key for the Codex CLI                           |
| `OPENAI_API_KEY`             | Fallback API key for the Codex CLI                  |

Provider credentials are passed through automatically from the host:
* The `anthropic` provider  forwards `CLAUDE_CODE_OAUTH_TOKEN` / `ANTHROPIC_API_KEY`, and 
* The `openai` provider forwards `CODEX_API_KEY` / `OPENAI_API_KEY`. 

Any value present in the host environment is injected when set.

## Entrypoint

The bundled `entrypoint.sh` sets up git and runs the agent in a retry loop, using `TASK_PROMPT` 
to tell the agent what to do. It branches on `AGENT_PROVIDER` to decide which agent binary to execute. 
It reads all configuration from environment variables.

Custom images can ignore this entrypoint entirely and define their own loop.

