// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"flag"
	"log/slog"
	"path/filepath"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/cli/batchinfo"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// cmdPatches exports commits from a completed workspace as patch files.
//
// The patches are written to a directory specified by dir.
// Supports a single workspace (nameOrID) or a batch of workspaces (batchID).
// Each workspace's patches are written to a subdirectory named after its branch.
//
// See cmdCollect for an alternative that cherry-picks commits directly
// into a target repository.
//
// The exportPatch function is injected to enable dependency-free testing.
type cmdPatches struct {
	nameOrID    string
	batchID     string
	dir         string
	exportPatch func(repoDir, outDir string) (int, error)
}

// newCmdPatches creates a new cmdPatches with the default export function.
// Test code creates cmdPatches directly to replace exportPatch with a mock.
func newCmdPatches() *cmdPatches {
	return &cmdPatches{exportPatch: workspace.ExportPatches}
}

// Name returns the subcommand name.
func (c *cmdPatches) Name() string { return "patches" }
func (c *cmdPatches) Desc() string { return "Export patches to directory" }
func (c *cmdPatches) Args() string { return "<name|id> --dir <dir>" }

// Parse expects the arguments for the subcommand.
// Requires dir for the output directory.
// In single mode, a workspace name or ID is required as a positional argument.
// In batch mode, batch selects all workspaces from a previously executed batch.
func (c *cmdPatches) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.StringVar(&c.batchID, "batch", "", "export patches for all workspaces in a batch")
		fs.StringVar(&c.dir, "dir", "", "output directory for patch files")
	})
	if err != nil {
		return err
	}
	if c.dir == "" {
		return cli.CmdErr(c, "%w", &cli.ArgMissingError{Name: "--dir"})
	}
	if c.batchID != "" {
		return nil
	}
	if fs.NArg() == 0 {
		return cli.CmdErr(c, "%w", &cli.ArgMissingError{Name: "name or id"})
	}
	c.nameOrID = fs.Arg(0)
	return nil
}

// Run exports patches from a single workspace or from all workspaces in a batch.
func (c *cmdPatches) Run(e *cli.Env) error {
	if c.batchID != "" {
		return c.runBatch(e)
	}
	return c.runSingle(e)
}

// runSingle exports patches from a single completed workspace into the output directory.
func (c *cmdPatches) runSingle(e *cli.Env) error {
	slog.Info("cli: patches", "nameOrID", c.nameOrID, "dir", c.dir)

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.ResolveCompleted(c.nameOrID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	n, err := c.exportPatch(ws.WorkDir(), c.dir)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	if n == 0 {
		e.Print("No patches to export from workspace %s.\n", ws.Name)
		return nil
	}

	e.Print("Exported %d patch(es) from %s into %s\n", n, ws.Name, c.dir)
	return nil
}

// runBatch exports patches from each workspace in a batch into per-branch subdirectories.
// Each result is collected as a group of patches saved to dir.
func (c *cmdPatches) runBatch(e *cli.Env) error {
	slog.Info("cli: patches-batch", "batchID", c.batchID, "dir", c.dir)

	rec, err := batch.LoadRecord(e.DataDir(), c.batchID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	results, err := batchinfo.Collect(e, rec, func(ws *workspace.Workspace, entry batch.Entry) (int, error) {
		return c.exportPatch(ws.WorkDir(), filepath.Join(c.dir, entry.Branch))
	})
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ok, failed := batchinfo.Report(e, results, "patch(es)")

	e.Print("\nExported patches for %d/%d workspaces into %s\n", ok, len(results), c.dir)

	if failed > 0 {
		return cli.CmdErr(c, "%d workspace(s) failed", failed)
	}

	return nil
}
