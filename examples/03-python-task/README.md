# 03: Python Task

Run a Python task using the `dwoe-agent:python` image.

```bash
cd examples/03-python-task
dwoe run task.yaml
```

## What It Does?

1. Uses `dwoe-agent:python` image (Python 3.12 + UV + Claude Code)
2. Copies the empty `./repo` into the workspace
3. The agent creates all files from scratch based on `prompt.md`
4. Results are committed inside the container

## Results

```bash
dwoe inspect <workspace-name>
dwoe collect --repo ./repo --branch agent-work <workspace-name>
```

## Files

* `task.yaml` task config pointing to the Python image
* `prompt.md` instructions to create a calc module with tests
* `repo/` empty source directory
