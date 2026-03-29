# 04: Batch Parallel

Run three tasks on the same Go repo in parallel, each in its own
isolated container.

```bash
cd examples/04-batch-parallel
dwoe batch .
```

## What It Does?

1. Discovers `task-greet.yaml`, `task-math.yaml`, `task-reverse.yaml`
2. Each task gets its own copy of `./repo` in an isolated workspace
3. Three containers start in parallel, each with its own prompt
4. Agents work independently and commit results
5. `dwoe collect --batch <id>` gathers all branches

## Results

Collect each agent's work as a branch in the source repo:

```bash
dwoe collect --batch <batch-id>
```

Or export as patch files instead:

```bash
dwoe patches --batch <batch-id> --dir ./patches
```

## Files

* `task-greet.yaml` task to add a Greet function
* `task-math.yaml` task to add Add and Multiply functions
* `task-reverse.yaml` task to add a Reverse function
* `prompts/` one prompt file per task
* `repo/` shared Go project source
