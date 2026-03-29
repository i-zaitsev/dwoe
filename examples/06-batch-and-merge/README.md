# 06: Batch and Merge

The full workflow: run tasks in parallel, collect patches, then use
another agent to merge everything together.

```bash
cd examples/06-batch-and-merge
dwoe batch .
```

## What It Does?

1. Two agents start in parallel (HTTP server + CLI flag parser)
2. After completion, patches are exported per task
3. Patches are copied into the repo for a merge agent
4. A `fire --do` agent applies and merges all patches
5. The merged result is collected as a branch

## Results

Export patches, then fire a merge agent:

```bash
dwoe patches --batch <batch-id> --dir ./patches
cp -r ./patches ./repo/patches

dwoe fire --repo ./repo \
  --do "Apply the git patches from the patches/ directory. \
Resolve any conflicts. Run go test ./... to verify everything \
works together. Commit the merged result."
```

Collect the merged result:

```bash
dwoe collect --repo ./repo --branch merged <merge-workspace>
```

## Files

* `task-api.yaml` task to add HTTP server package
* `task-cli.yaml` task to add CLI flag parser package
* `prompts/` one prompt file per task
* `repo/` shared Go project source
