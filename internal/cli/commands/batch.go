// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// cmdBatch runs a group of workers using task-*.yaml files defined in the dir.
//
// The subcommand runs multiple tasks in separate workspaces. Each workspace is a standalone
// copy/checkout of the working repository. The command starts workers and waits for results.
// At the end of waiting, it reports the number of succeeded and failed workers.

// The loadConfig and ensureRepo injected to enable dependency-free testing.
// The loadConfig function reads each task file from the dir and ensures they are valid.
// The ensureRepo function checks if the local repo configured in the task is ready for work.
type cmdBatch struct {
	dir        string
	loadConfig func(taskPath, dataDir string) (*config.Task, error)
	ensureRepo func(repoPath, userName, userEmail string) error
}

// newCmdBatch creates an instance of cmdBatch with injected dependencies.
// Test code creates cmdBatch directly to replace the real functionality with mocks.
func newCmdBatch() *cmdBatch {
	return &cmdBatch{
		loadConfig: config.LoadMergedConfig,
		ensureRepo: workspace.EnsureRepoReady,
	}
}

// Name returns the name of a command.
func (c *cmdBatch) Name() string { return "batch" }
func (c *cmdBatch) Desc() string { return "Run all task files in parallel" }
func (c *cmdBatch) Args() string { return "<dir>" }

// Parse expects the arguments for the command.
// The batch command requires only a single positional parameter providing the dir with tasks.
func (c *cmdBatch) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, nil)
	if err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return cli.CmdErr(c, "%w", &cli.ArgMissingError{Name: "directory"})
	}
	c.dir = fs.Arg(0)
	return nil
}

// Run launches multiple workers defined in the batch dir.
// Each worker is expected to implement a single task in the configured repository.
// At the moment, it is assumed that all tasks work on the same repository.
func (c *cmdBatch) Run(e *cli.Env) error {
	slog.Info("cli: batch", "dir", c.dir)

	taskFiles, err := discoverTasks(c.dir)
	if err != nil {
		return err
	}

	slog.Debug("cli: batch", "files", len(taskFiles))

	e.Print("batch: discovered %d task(s) in %s\n", len(taskFiles), c.dir)

	slog.Debug("cli: batch", "phase", "resolving source repo")
	sourceDir, err := c.resolveBatchRepo(taskFiles, e.DataDir(), e.SourceDir())
	if err != nil {
		return err
	}

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	slog.Debug("cli: batch", "phase", "starting tasks")
	ids, err := c.startAll(e, taskFiles)
	if err != nil {
		return err
	}

	slog.Debug("cli: batch saving record", "dataDir", e.DataDir())
	relPaths := make([]string, len(taskFiles))
	for i, tf := range taskFiles {
		rel, errRel := filepath.Rel(c.dir, tf)
		if errRel != nil {
			rel = tf
		}
		relPaths[i] = rel
	}
	rec := batch.NewRecord(sourceDir, relPaths, ids)
	if errSave := batch.SaveRecord(e.DataDir(), rec); errSave != nil {
		return cli.CmdErr(c, "%w", errSave)
	}

	e.Print("Batch ID: %s\n", rec.ID)

	return c.waitAll(e, manager, ids)
}

// resolveBatchRepo ensures that the repo used by each batch task is ready.
// It is assumed that all tasks in the batch use the same repo.
func (c *cmdBatch) resolveBatchRepo(taskFiles []string, dataDir, sourceDir string) (string, error) {
	var (
		repoPath string
		gitUser  config.GitUser
	)
	for _, taskFile := range taskFiles {
		cfg, err := c.loadConfig(taskFile, dataDir)
		if err != nil {
			return "", cli.CmdErr(c, "task file %s: %w", taskFile, err)
		}
		cfg.FallbackSource(sourceDir)
		taskRepoPath := cfg.Source.LocalPath
		gitUser.Name = cfg.Git.Name
		gitUser.Email = cfg.Git.Email
		if repoPath == "" {
			repoPath = taskRepoPath
		} else if repoPath != taskRepoPath {
			return "", cli.CmdErr(c, "subtask repo name mismatch: %s", taskRepoPath)
		}
	}
	if repoPath == "" {
		return "", cli.CmdErr(c, "source repo is not found")
	}
	errReady := c.ensureRepo(repoPath, gitUser.Name, gitUser.Email)
	if errReady != nil {
		return "", errReady
	}
	return repoPath, nil
}

// startAll starts all tasks defined in the dir.
// Each created workspace is started in a detached mode.
// The created IDs are returned to the caller.
func (c *cmdBatch) startAll(e *cli.Env, taskFiles []string) ([]string, error) {
	slog.Debug("cli: batch", "phase", "start-all")
	var ids []string
	for _, tf := range taskFiles {
		cmd := &cmdRun{taskPath: tf, detach: true}
		if err := cmd.Run(e); err != nil {
			if errors.Is(err, workspace.ErrWorkspaceDone) {
				continue
			}
			return nil, cli.CmdErr(c, "%w", err)
		}
		ids = append(ids, cmd.createdID)
	}
	return ids, nil
}

// waitAll waits for all tasks to complete.
// The started tasks are running in detached mode. The waiting function creates a polling goroutine
// for each task. The goroutine blocks until the task completes.
func (c *cmdBatch) waitAll(e *cli.Env, manager *workspace.Manager, ids []string) error {
	e.Print("\nWaiting for %d workspace(s)...\n", len(ids))

	type result struct {
		name   string
		status string
	}
	results := make([]result, len(ids))

	var wg sync.WaitGroup
	slog.Debug("cli: batch wait all", "total", len(ids))
	for i, id := range ids {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			ws, err := manager.Get(id)
			if err != nil {
				results[idx] = result{name: id, status: workspace.StatusFailed}
				return
			}
			exitCode, waitErr := manager.Wait(e.Context(), ws.ID)
			status := workspace.StatusCompleted
			if waitErr != nil || exitCode != 0 {
				status = workspace.StatusFailed
			}
			if err := manager.Cleanup(context.WithoutCancel(e.Context()), ws.ID); err != nil {
				slog.Warn("batch: cleanup", "id", ws.ID, "err", err)
			}
			results[idx] = result{name: ws.Name, status: status}
			e.Print("  %-24s %s (exit %d)\n", ws.Name, status, exitCode)
		}(i, id)
	}
	wg.Wait()

	completed, failed := 0, 0
	for _, r := range results {
		if r.status == workspace.StatusCompleted {
			completed++
		} else {
			failed++
		}
	}

	slog.Debug("cli: batch wait all", "completed", completed, "failed", failed)

	e.Print("\nSummary: %d total, %d completed, %d failed\n", len(results), completed, failed)

	if failed > 0 {
		return cli.CmdErr(c, "%d of %d workspace(s) failed", failed, len(results))
	}

	return nil
}

// discoverTasks walks dir recursively and returns all .yaml files found.
// The returned paths are sorted lexicographically.
func discoverTasks(dir string) ([]string, error) {
	var tasks []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		tasks = append(tasks, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("batch: walk: %w", err)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("batch: no .yaml task files found in %s", dir)
	}
	sort.Strings(tasks)
	return tasks, nil
}
