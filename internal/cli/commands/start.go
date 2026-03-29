// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/cli"
)

// cmdStart starts a previously created workspace.
//
// The workspace must exist and be in the pending state. Starting a workspace
// provisions Docker resources (network, containers) and launches the agent.
type cmdStart struct {
	nameOrID string
}

// Name returns the subcommand name.
func (c *cmdStart) Name() string { return "start" }
func (c *cmdStart) Desc() string { return "Start existing workspace" }
func (c *cmdStart) Args() string { return "<name|id>" }

// Parse expects the arguments for the subcommand.
// Requires a workspace name or ID as the first positional argument.
func (c *cmdStart) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, nil)
	if err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return cli.CmdErr(c, "%w", &cli.ArgMissingError{Name: "name or id"})
	}
	c.nameOrID = fs.Arg(0)
	return nil
}

// Run starts the workspace identified by name or ID.
func (c *cmdStart) Run(e *cli.Env) error {
	slog.Info("cli: start", "nameOrID", c.nameOrID)

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.Resolve(c.nameOrID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	if err := manager.Start(e.Context(), ws.ID); err != nil {
		return err
	}

	e.Print("Started workspace: %s\n", ws.Name)
	e.Print("- ID: %s\n", ws.ID)
	e.Print("- Status: running\n")
	return nil
}
