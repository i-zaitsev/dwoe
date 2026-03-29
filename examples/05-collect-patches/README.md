# 05: Collect Patches

Run a task, then export the agent's commits as portable patch files.

```bash
cd examples/05-collect-patches
dwoe run task.yaml
```

## What It Does?

1. Creates a workspace and runs the agent on `./repo`
2. The agent implements the task and commits results
3. After completion, patches are exported with `dwoe patches`
4. Patches can be reviewed, shared, or applied to any repo clone

## Results

Export patches after the workspace completes:

```bash
dwoe patches --dir ./patches <workspace-name>
```

Apply patches elsewhere:

```bash
cd /path/to/your/repo
git am /path/to/patches/*.patch
```

## Files

* `task.yaml` task configuration
* `prompt.md` instructions to add a slugify function
* `repo/` Go project source
