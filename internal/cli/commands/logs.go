// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"bufio"
	"context"
	"flag"
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// cmdLogs streams the container logs of a running workspace.
//
// The command attaches to the agent container's log output and prints each
// line to stdout until the container exits or the context is cancelled.
type cmdLogs struct {
	nameOrID string
	follow   bool
}

// Name returns the subcommand name.
func (c *cmdLogs) Name() string { return "logs" }
func (c *cmdLogs) Desc() string { return "Show workspace logs" }
func (c *cmdLogs) Args() string { return "<name|id>" }

// Parse expects the arguments for the subcommand.
// Requires a workspace name or ID as a positional argument. The follow flag
// controls whether to stream logs continuously (default: true).
func (c *cmdLogs) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.BoolVar(&c.follow, "follow", false, "follow the logs from container")
		fs.BoolVar(&c.follow, "f", false, "follow the logs from container")
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

func (c *cmdLogs) Run(e *cli.Env) error {
	slog.Info("cli: logs", "nameOrID", c.nameOrID, "follow", c.follow)

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.Resolve(c.nameOrID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	logs, err := manager.Logs(e.Context(), ws.ID, c.follow)
	if err != nil {
		if ws.Status == workspace.StatusCompleted || ws.Status == workspace.StatusFailed {
			return cli.CmdErr(c, "workspace destroyed, logs unavailable")
		}
		return err
	}
	defer logs.Close()

	ctx := e.Context()
	stop := context.AfterFunc(ctx, func() { logs.Close() })
	defer stop()

	const (
		kb = 1024
		mb = 1024 * kb
	)
	scanner := bufio.NewScanner(logs)
	scanner.Buffer(make([]byte, 0, 64*kb), mb)

	for scanner.Scan() {
		e.Print("%s\n", scanner.Text())
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		slog.Warn("logs: scanner", "err", err)
	}
	return nil
}
