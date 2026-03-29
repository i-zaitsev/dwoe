// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"errors"
	"flag"
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// errWsRunning is returned when attempting to destroy
// a running workspace without the force flag.
var errWsRunning = errors.New("workspace is running")

// cmdDestroy removes a workspace and all its associated resources.
//
// The command stops and removes containers, networks, and deletes the workspace
// directory from disk. By default, it refuses to destroy a running workspace;
// use force to override. The all flag destroys every workspace.
type cmdDestroy struct {
	nameOrID string
	force    bool
	all      bool
}

// Name returns the subcommand name.
func (c *cmdDestroy) Name() string { return "destroy" }
func (c *cmdDestroy) Desc() string { return "Remove workspace" }
func (c *cmdDestroy) Args() string { return "<name|id>" }

// Parse expects the arguments for the subcommand.
// Requires a workspace name or ID as a positional argument, or all to target
// every workspace. The force flag allows destroying running workspaces.
func (c *cmdDestroy) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.BoolVar(&c.force, "force", false, "destroy even if running")
		fs.BoolVar(&c.force, "f", false, "destroy even if running")
		fs.BoolVar(&c.all, "all", false, "destroy all workspaces")
	})
	if err != nil {
		return err
	}
	if !c.all && fs.NArg() == 0 {
		return cli.CmdErr(c, "%w", &cli.ArgMissingError{Name: "name or id"})
	}
	if fs.NArg() > 0 {
		c.nameOrID = fs.Arg(0)
	}
	return nil
}

// Run destroys a single workspace or all workspaces when all is set.
// A running workspace is rejected unless force is specified.
func (c *cmdDestroy) Run(e *cli.Env) error {
	slog.Info("cli: destroy", "nameOrID", c.nameOrID, "force", c.force, "all", c.all)

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	if c.all {
		if !c.force {
			return cli.CmdErr(c, "--all: %w", &cli.ArgMissingError{Name: "--force"})
		}
		return c.destroyAll(e, manager)
	}

	ws, err := manager.Resolve(c.nameOrID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	if ws.Status == workspace.StatusRunning && !c.force {
		return cli.CmdErr(c, "%w", errWsRunning)
	}

	e.Print("Destroying workspace: %s\n", ws.Name)
	if err := manager.Destroy(e.Context(), ws.ID, workspace.DestroyOpts{}); err != nil {
		return err
	}
	e.Print("Workspace destroyed.\n")
	return nil
}

// destroyAll iterates over every workspace and destroys each one.
func (c *cmdDestroy) destroyAll(e *cli.Env, manager *workspace.Manager) error {
	wss, err := manager.List()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}
	if len(wss) == 0 {
		e.Print("No workspaces to destroy.\n")
		return nil
	}
	for _, ws := range wss {
		e.Print("Destroying workspace: %s\n", ws.Name)
		if err := manager.Destroy(e.Context(), ws.ID, workspace.DestroyOpts{}); err != nil {
			return err
		}
		e.Print("Workspace destroyed.\n")
	}
	return nil
}
