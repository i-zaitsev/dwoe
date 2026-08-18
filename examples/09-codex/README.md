# 09: Codex (OpenAI)

Run a task with the OpenAI Codex CLI instead of Claude by setting `provider: openai`.

The images are provider-agnostic: this example uses the same `dwoe-agent:python` image
that a Claude task would use, because every agent image ships both CLIs. Only
`agent.provider` decides which one runs.

```bash
make image-python
export OPENAI_API_KEY=sk-...
cd examples/09-codex
dwoe run task.yaml
```

## What It Does?

1. Creates a workspace from the task configuration
2. Copies `./repo` into the workspace as `/workspace`
3. Copies `prompt.md` alongside the source code
4. Starts the Codex agent (`codex exec`) and follows logs until the container exits

`OPENAI_API_KEY` is passed through from your host environment automatically; no
`env_vars` block is required.

## Other OpenAI-compatible models

The `openai` provider also drives any OpenAI-compatible endpoint (self-hosted
vLLM/TGI, or hosted gateways). Point Codex at the endpoint by exporting
`OPENAI_BASE_URL` and setting it through `agent.env_vars`:

```yaml
agent:
  provider: openai
  image: dwoe-agent:python
  model: <model-id>
  env_vars:
    OPENAI_BASE_URL: ${OPENAI_BASE_URL}
    OPENAI_API_KEY: ${OPENAI_API_KEY}
```

Make sure the endpoint's host is in the proxy allowlist (or run with `--noproxy`).

## Results

```bash
dwoe inspect <workspace-name>
dwoe collect --repo ./repo --branch agent-work <workspace-name>
```

## Files

* `task.yaml` full task configuration
* `prompt.md` instructions for the agent
* `repo/` Python project source code
