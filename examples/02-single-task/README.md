# 02: Single Task (Go)

Run a single task with full control via `task.yaml`.

```bash
cd examples/02-single-task
dwoe run task.yaml
```

## What It Does?

1. Creates a workspace from the task configuration
2. Copies `./repo` into the workspace as `/workspace`
3. Copies `prompt.md` alongside the source code
4. Starts the agent and follows logs until the container exits

## Results

```bash
dwoe inspect <workspace-name>
dwoe collect --repo ./repo --branch agent-work <workspace-name>
```

## Resuming

To resume the workspace instead of creating a new one each time, add
`continue_policy: resume` to `task.yaml`. The agent picks up where
it left off on the same filesystem.

## Files

* `task.yaml` full task configuration
* `prompt.md` instructions for the agent
* `repo/` Go project source code
