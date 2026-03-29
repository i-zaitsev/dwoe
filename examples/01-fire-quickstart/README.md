# 01: Fire Quickstart

The simplest way to run `dwoe` which requires no `task.yaml` file is 
using the `fire` command.

```bash
cd examples/01-fire-quickstart
dwoe fire --repo ./src --work task.md
# Started workspace: steep-steady-quail
# - ID: 2a4d1050-65ba-41db-bdcb-e77a546676e2
# View logs: dwoe logs 2a4d1050-65ba-41db-bdcb-e77a546676e2
```

## What It Does?

1. The `./src` folder is copied into a new workspace as `/workspace`
2. The `task.md` file is copied alongside the source code
3. The agent reads all files in `/workspace` and follows the instructions
4. Results are committed inside the container

## Results

Inspecting logs and results when the task is completed.

```bash
dwoe logs <workspace-name>
dwoe inspect <workspace-name>
```

## Files

* `task.md` stores instructions for the agent
* `src/` contains source code for the agent to work on
