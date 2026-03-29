# 07: Custom Prompt and No-Proxy

Define the agent prompt inline in `task.yaml` and run without a proxy
container.

```bash
cd examples/07-custom-prompt
dwoe run task.yaml
```

## What It Does?

1. Uses `task_prompt` in YAML instead of a separate prompt file
2. Skips the proxy container with `no_proxy: true`
3. The agent has unrestricted network access
4. Results are committed inside the container

## Results

```bash
dwoe inspect <workspace-name>
dwoe logs <workspace-name>
```

## Files

* `task.yaml` config with inline prompt and no-proxy
* `repo/` simple Go project to modify
