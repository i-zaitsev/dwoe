// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/cli"
)

// cmdStatus displays the current state of a workspace.
//
// The output includes the workspace name, ID, status, and timestamps for
// creation, start, and finish events.
type cmdStatus struct {
	nameOrID string
}

// Name returns the subcommand name.
func (c *cmdStatus) Name() string { return "status" }
func (c *cmdStatus) Desc() string { return "Show workspace status" }
func (c *cmdStatus) Args() string { return "<name|id>" }

// Parse expects the arguments for the subcommand.
// Requires a workspace name or ID as the first positional argument.
func (c *cmdStatus) Parse(args []string) error {
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

// Run prints the workspace details to stdout.
func (c *cmdStatus) Run(e *cli.Env) error {
	slog.Info("cli: status", "nameOrID", c.nameOrID)

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.Resolve(c.nameOrID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	e.Print("Name:     %s\n", ws.Name)
	e.Print("ID:       %s\n", ws.ID)
	e.Print("Status:   %s\n", ws.Status)
	e.Print("Created:  %s\n", cli.FmtTime(ws.CreatedAt))
	e.Print("Started:  %s\n", cli.FmtTime(ws.StartedAt))
	e.Print("Finished: %s\n", cli.FmtTime(ws.FinishedAt))
	e.Print("Exit:     %s\n", ws.ExitStatus())
	return nil
}
