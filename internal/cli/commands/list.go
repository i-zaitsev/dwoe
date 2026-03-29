// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"text/tabwriter"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// cmdList displays all known workspaces.
//
// The format flag controls the output format.
// Supported formats are "table" (default) and "json".
type cmdList struct {
	format string
}

// Name returns the subcommand name.
func (c *cmdList) Name() string { return "list" }
func (c *cmdList) Desc() string { return "List workspaces" }
func (c *cmdList) Args() string { return "[--format FMT]" }

// Parse expects the arguments for the subcommand.
// The optional format flag selects the output format: "table" or "json".
func (c *cmdList) Parse(args []string) error {
	_, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.StringVar(&c.format, "format", "table", "output format (table, json)")
	})
	if err != nil {
		return err
	}
	if c.format != "table" && c.format != "json" {
		return cli.CmdErr(c, "%w", &cli.ArgInvalidError{Name: "format", Value: c.format})
	}
	return nil
}

// Run lists all workspaces in the configured data directory.
func (c *cmdList) Run(e *cli.Env) error {
	slog.Info("cli: list", "format", c.format)

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	workspaces, err := manager.List()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	if c.format == "table" {
		return printTable(e, workspaces)
	}

	return printJSON(e, workspaces)
}

// printTable writes workspaces as a tab-aligned table to stdout.
func printTable(e *cli.Env, wss []*workspace.Workspace) error {
	if len(wss) == 0 {
		e.Print("No workspaces found.\n")
		return nil
	}
	tw := tabwriter.NewWriter(e.Stdout(), 0, 4, 2, ' ', tabwriter.TabIndent)
	_, _ = fmt.Fprintln(tw, "NAME\tID\tSTATUS\tEXIT\tCREATED\tSTARTED\tFINISHED")
	for _, ws := range wss {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ws.Name,
			cli.CutIfLong(ws.ID),
			ws.Status,
			ws.ExitStatus(),
			cli.FmtTime(ws.CreatedAt),
			cli.FmtTime(ws.StartedAt),
			cli.FmtTime(ws.FinishedAt),
		)
	}
	return tw.Flush()
}

// printJSON writes workspaces as indented JSON to stdout.
func printJSON(e *cli.Env, wss []*workspace.Workspace) error {
	enc := json.NewEncoder(e.Stdout())
	enc.SetIndent("", "  ")
	return enc.Encode(wss)
}
