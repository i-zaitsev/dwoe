// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"flag"
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/cli/batchinfo"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// cmdCollect cherry-picks commits from a workspace into a target repo.
//
// The commits from a completed workspace are cherry-picked into a target git
// repository on a new branch. Supports a single task/workspace (by name/id)
// or a batch of tasks (by batch id). Each task is expected to work on a
// separate branch, which is copied directly into the target repository.
//
// See commands.cmdPatches for a more granular approach that collects the
// work as a list of patch files.
//
// The collect function is injected to enable dependency-free testing.
type cmdCollect struct {
	nameOrID string
	repo     string
	branch   string
	batchID  string
	collect  func(workspaceDir, targetRepo, branch string) (int, error)
}

// newCmdCollect creates a new cmdCollect.
// Test code creates cmdCollect directly to replace the collect function with a mock.
func newCmdCollect() *cmdCollect {
	return &cmdCollect{collect: workspace.Collect}
}

// Name returns the subcommand name.
func (c *cmdCollect) Name() string { return "collect" }
func (c *cmdCollect) Desc() string { return "Collect commits into a repo branch" }
func (c *cmdCollect) Args() string { return "<name|id> [--batch ID]" }

// Parse parses the subcommand arguments.
// Requires repo and branch for a single mode, or batch for batch mode.
// The repo mode takes a single branch from a worker.
// The batch mode locates the previously executed batch tasks and collects all the branches from there.
func (c *cmdCollect) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.StringVar(&c.repo, "repo", "", "path to target git repository")
		fs.StringVar(&c.branch, "branch", "", "branch name to create")
		fs.StringVar(&c.batchID, "batch", "", "collect all workspaces from a batch")
	})
	if err != nil {
		return err
	}
	if c.batchID != "" {
		return nil
	}
	if c.repo == "" {
		return cli.CmdErr(c, "flag: %w", &cli.ArgMissingError{Name: "--repo"})
	}
	if c.branch == "" {
		return cli.CmdErr(c, "flag: %w", &cli.ArgMissingError{Name: "--branch"})
	}
	if fs.NArg() == 0 {
		return cli.CmdErr(c, "arg: %w", &cli.ArgMissingError{Name: "name or id"})
	}
	c.nameOrID = fs.Arg(0)
	return nil
}

// Run collects commits from a single workspace or from all workspaces in a batch.
func (c *cmdCollect) Run(e *cli.Env) error {
	if c.batchID != "" {
		return c.runBatch(e)
	}
	return c.runSingle(e)
}

// runSingle collects results from a single worker.
// The commits are collected as a new branch in the target repository.
func (c *cmdCollect) runSingle(e *cli.Env) error {
	slog.Info("cli: collect-single", "nameOrID", c.nameOrID, "repo", c.repo, "branch", c.branch)

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.ResolveCompleted(c.nameOrID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	n, err := c.collect(ws.WorkDir(), c.repo, c.branch)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	if n == 0 {
		e.Print("No commits to collect from workspace %s.\n", ws.Name)
	} else {
		e.Print("Collected %d commit(s) from %s into branch %s\n", n, ws.Name, c.branch)
	}

	return nil
}

// runBatch collects results from all workspaces in a batch.
// Each result is collected as a new branch in the target repository.
// Note that the command does not attempt to merge commits. They are simply added to the target.
func (c *cmdCollect) runBatch(e *cli.Env) error {
	slog.Info("cli: collect-batch", "batchID", c.batchID)

	rec, err := batch.LoadRecord(e.DataDir(), c.batchID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	e.Print("Started collecting workspaces:\n")

	results, err := batchinfo.Collect(e, rec, func(ws *workspace.Workspace, entry batch.Entry) (int, error) {
		return c.collect(ws.WorkDir(), rec.SourceDir, entry.Branch)
	})
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ok, failed := batchinfo.Report(e, results, "commit(s)")

	e.Print("\nCollected %d/%d workspaces into %s\n", ok, len(results), rec.SourceDir)

	if failed > 0 {
		return cli.CmdErr(c, "%d workspace(s) failed", failed)
	}

	return nil
}
