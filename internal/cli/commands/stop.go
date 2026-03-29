// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"flag"
	"log/slog"
	"time"

	"github.com/i-zaitsev/dwoe/internal/cli"
)

// cmdStop stops a running workspace.
//
// By default, the agent container is given one minute to shut down gracefully.
// The force flag sends SIGKILL immediately with no grace period.
type cmdStop struct {
	nameOrID string
	force    bool
}

// Name returns the subcommand name.
func (c *cmdStop) Name() string { return "stop" }
func (c *cmdStop) Desc() string { return "Stop running workspace" }
func (c *cmdStop) Args() string { return "<name|id>" }

// Parse expects the arguments for the subcommand.
// Requires a workspace name or ID as the first positional argument. The
// force flag skips the graceful shutdown period.
func (c *cmdStop) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.BoolVar(&c.force, "force", false, "force kill (SIGKILL)")
		fs.BoolVar(&c.force, "f", false, "force kill (SIGKILL)")
	})
	if err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return cli.CmdErr(c, "%w", &cli.ArgMissingError{Name: "name or id"})
	}
	c.nameOrID = fs.Arg(0)
	return nil
}

// Run stops the workspace's agent container.
func (c *cmdStop) Run(e *cli.Env) error {
	slog.Info("cli: stop", "nameOrID", c.nameOrID, "force", c.force)

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.Resolve(c.nameOrID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	timeout := time.Minute
	if c.force {
		timeout = 0
	}

	e.Print("Stopping workspace: %s\n", ws.Name)
	if err := manager.Stop(e.Context(), ws.ID, timeout); err != nil {
		return err
	}
	e.Print("Workspace stopped.\n")
	return nil
}
