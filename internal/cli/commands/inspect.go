// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/cli"
)

type cmdInspect struct {
	nameOrID string
}

func (c *cmdInspect) Name() string { return "inspect" }
func (c *cmdInspect) Desc() string { return "Show detailed workspace info" }
func (c *cmdInspect) Args() string { return "<name|id>" }

func (c *cmdInspect) Parse(args []string) error {
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

func (c *cmdInspect) Run(e *cli.Env) error {
	slog.Info("cli: inspect", "nameOrID", c.nameOrID)

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.Resolve(c.nameOrID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	e.Print("Name:       %s\n", ws.Name)
	e.Print("ID:         %s\n", ws.ID)
	e.Print("Status:     %s\n", ws.Status)
	e.Print("BasePath:   %s\n", ws.BasePath)
	e.Print("Agent:      %s\n", containerOrNone(ws.ContainerIDs, "agent"))
	e.Print("Proxy:      %s\n", containerOrNone(ws.ContainerIDs, "proxy"))
	e.Print("NetworkID:  %s\n", ws.NetworkID)
	e.Print("Created:    %s\n", cli.FmtTime(ws.CreatedAt))
	e.Print("Started:    %s\n", cli.FmtTime(ws.StartedAt))
	e.Print("Finished:   %s\n", cli.FmtTime(ws.FinishedAt))

	return nil
}

func containerOrNone(ids map[string]string, key string) string {
	if v := ids[key]; v != "" {
		return v
	}
	return "(none)"
}
