# Examples

Each example demonstrates a different way to use `dwoe`.

## Prerequisites

* Docker running locally
* A Claude Code OAuth token: `claude setup-token`
* Agent image built from `docker/` (or any custom image with compatible entrypoint):

```bash
docker build -f docker/Dockerfile.base -t dwoe-agent:latest docker/
docker build -f docker/Dockerfile.claude-go -t dwoe-agent:go docker/
docker build -f docker/Dockerfile.claude-python -t dwoe-agent:python docker/
docker build -f docker/Dockerfile.proxy -t dwoe-proxy:latest docker/
```

* Setting up the token in the shell before running agents:

```bash
export CLAUDE_CODE_OAUTH_TOKEN=sk-ant-oat01-...
```

## Usage Examples

| # | Directory             | Command                    | What it shows                                   |
|---|-----------------------|----------------------------|-------------------------------------------------|
| 1 | `01-fire-quickstart/` | `dwoe fire`                | Quickest start: one command, minimal config     |
| 2 | `02-single-task/`     | `dwoe run`                 | Full task.yaml config with a Go project         |
| 3 | `03-python-task/`     | `dwoe run`                 | Python project with a different image           |
| 4 | `04-batch-parallel/`  | `dwoe batch`               | Three tasks on the same repo in parallel        |
| 5 | `05-collect-patches/` | `dwoe patches`             | Export agent commits as patch files             |
| 6 | `06-batch-and-merge/` | `dwoe batch` + `fire --do` | Full workflow: parallel tasks → patches → merge |
| 7 | `07-custom-prompt/`   | `dwoe run`                 | Custom agent prompt and no-proxy mode           |

## Workspace Layout

When `dwoe` starts a task, it creates a workspace under `~/.dwoe/workspaces/{id}/`:

```
{id}/
├── CLAUDE.md               # agent guidelines
├── config.yaml             # saved task config
├── logs
│   ├── agent               # /logs in the container
│   │   └── container.log
│   └── proxy               # /var/log/squid in the proxy container
│       └── access.log
├── proxy
│   ├── allowlist.txt       # (ro) /etc/squid/squid.conf
│   └── squid.conf          # (ro) /etc/squid/allowlist.txt
├── settings.json           # (ro) Claude Code permissions
└── workspace               # /workspace in the container
    ├── CLAUDE.md
    ├── PROGRESS.md
    ├── hello.py
    └── task.md
```

The location of workspaces can be changed with `datadir` config option.

## Commands: `run` vs `fire`

* `fire` is the quick entry point requiring a repo and task file (`--work`) or an inline prompt (`--do`)
* `run` takes a full `task.yaml` for complete control over image, model, resources, proxy, and environment
