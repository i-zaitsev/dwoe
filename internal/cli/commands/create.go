// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"flag"
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/config"
)

// cmdCreate creates a new workspace.
// The workspace is not started. Only the metadata and related files are created/copied.
// Use commands.cmdStart to start the workspace.
type cmdCreate struct {
	taskPath string
	name     string
}

// Name returns the subcommand name.
func (c *cmdCreate) Name() string { return "create" }
func (c *cmdCreate) Desc() string { return "Create workspace from config" }
func (c *cmdCreate) Args() string { return "<task.yaml>" }

// Parse expects the arguments for the subcommand.
// The subcommand requires a task file as the first argument.
// Optional argument overrides the workspace name from the task file.
func (c *cmdCreate) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.StringVar(&c.name, "name", "", "override workspace name")
	})
	if err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return cli.CmdErr(c, "%w", &cli.ArgMissingError{Name: "task file"})
	}
	c.taskPath = fs.Arg(0)
	return nil
}

// Run creates a new workspace.
// It starts in the pending state.
func (c *cmdCreate) Run(e *cli.Env) error {
	slog.Info("cli: create", "taskPath", c.taskPath, "name", c.name)

	taskCfg, err := config.LoadMergedConfig(c.taskPath, e.DataDir())
	if err != nil {
		return cli.CmdErr(c, "load config: %w", err)
	}
	if e.NoProxy() {
		taskCfg.NoProxy = true
	}
	if c.name != "" {
		taskCfg.Name = c.name
	}

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.FindOrCreate(taskCfg)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	var verb string
	if taskCfg.PolicyRequiresNew() {
		verb = "created"
	} else {
		verb = "resumed"
	}

	slog.Info("workspace run", "name", ws.Name, "id", ws.ID, "verb", verb)

	e.Print("Workspace %s: %s\n", verb, ws.Name)
	e.Print("- ID: %s\n", ws.ID)
	e.Print("- Status: %s\n", ws.Status)
	e.Print("- Path: %s\n", ws.BasePath)
	e.Print("Run '%s start %s' to start the task.\n", cli.Prog, ws.Name)

	return nil
}
