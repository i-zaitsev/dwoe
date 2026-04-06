// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"time"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// exitError indicates that the workspace's agent container exited with a non-zero status.
type exitError struct {
	code int
}

// errRunInterrupted is returned when the run is cancelled by an OS signal.
var errRunInterrupted = errors.New("interrupted")

// Error returns a human-readable description of the exit failure.
func (e *exitError) Error() string {
	return fmt.Sprintf("workspace failed with exit code %d", e.code)
}

// cmdRun creates, starts, and optionally follows a workspace in one step.
//
// It combines the create, start, and logs commands into a single operation.
// In attached mode (default), it streams the agent's logs until the container
// exits and then reports the final status. In detached mode, it returns
// immediately after the workspace is started.
//
// The createdID field is set after a successful create and is used by cmdBatch
// to track which workspaces were started.
type cmdRun struct {
	taskPath  string
	name      string
	detach    bool
	createdID string
}

// Name returns the subcommand name.
func (c *cmdRun) Name() string { return "run" }
func (c *cmdRun) Desc() string { return "Create and start workspace" }
func (c *cmdRun) Args() string { return "<task.yaml>" }

// Parse expects the arguments for the subcommand.
// Requires a task file as the first positional argument. Optional flags
// override the workspace name or enable detached mode.
func (c *cmdRun) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.StringVar(&c.name, "name", "", "override workspace name")
		fs.BoolVar(&c.detach, "detach", false, "start and return immediately")
		fs.BoolVar(&c.detach, "d", false, "start and return immediately")
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

// Run creates a workspace, starts it, and either follows its logs or returns
// immediately depending on the detach flag.
func (c *cmdRun) Run(e *cli.Env) error {
	slog.Info("cli: run", "taskPath", c.taskPath, "name", c.name, "detach", c.detach)

	ctx := e.Context()

	slog.Debug("run: loading config", "path", c.taskPath)
	taskCfg, err := config.LoadMergedConfig(c.taskPath, e.DataDir())
	if err != nil {
		return cli.CmdErr(c, "load config: %w", err)
	}
	taskCfg.FallbackSource(e.SourceDir())
	if e.Model() != "" && taskCfg.Agent.Model == "" {
		taskCfg.Agent.Model = e.Model()
	}
	if e.NoProxy() {
		taskCfg.NoProxy = true
	}
	if c.name != "" {
		taskCfg.Name = c.name
	}

	slog.Debug("run: creating workspace", "dataDir", e.DataDir())
	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.FindOrCreate(taskCfg)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}
	c.createdID = ws.ID

	slog.Debug("run: starting workspace", "id", ws.ID)
	err = manager.Start(ctx, ws.ID)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	var verb string
	if taskCfg.PolicyRequiresNew() {
		verb = "started"
	} else {
		verb = "resumed"
	}
	e.Print("Workspace %s: %s\n", verb, ws.Name)
	e.Print("- ID: %s\n", ws.ID)
	e.Print("- Status: running\n")
	e.Print("- Path: %s\n", ws.BasePath)

	if c.detach {
		e.Print("View logs: %s logs %s\n", cli.Prog, ws.ID)
		return nil
	}

	slog.Debug("run: reading worker logs", "id", ws.ID)
	logs, err := manager.Logs(ctx, ws.ID, true)

	if err != nil {
		slog.Error("run: cannot read running job logs", "err", err)
		errStop := manager.Stop(context.Background(), ws.ID, time.Minute)
		e.Error("failed to read the logs from attached worker")
		if errStop != nil {
			return fmt.Errorf("fatal: cannot stop the workspace: %w", errStop)
		}
		return cli.CmdErr(c, "%w", err)
	}

	lines := make(chan string)
	logCtx, logCancel := context.WithCancel(ctx)
	go cli.ScanLogs(logCtx, logs, lines)

	for line := range lines {
		e.Print("%s\n", line)
	}

	exitCode, waitErr := manager.Wait(ctx, ws.ID)
	logCancel()

	if ctx.Err() != nil {
		e.Print("\nInterrupted. Stopping workspace %s...\n", ws.Name)
		bgCtx := context.Background()
		if errStop := manager.Stop(bgCtx, ws.ID, 30*time.Second); errStop != nil {
			slog.Error("run: stop on interrupt", "err", errStop)
		}
		if errCleanup := manager.Cleanup(bgCtx, ws.ID); errCleanup != nil {
			slog.Error("run: cleanup on interrupt", "err", errCleanup)
		}
		return cli.CmdErr(c, "%w", errRunInterrupted)
	}

	status := workspace.StatusCompleted
	if waitErr != nil || exitCode != 0 {
		status = workspace.StatusFailed
	}
	if errCleanup := manager.Cleanup(context.Background(), ws.ID); errCleanup != nil {
		slog.Error("run: cleanup", "err", errCleanup)
	}

	e.Print("Workspace %s: %s (exit code %d)\n", ws.Name, status, exitCode)
	if exitCode != 0 {
		return cli.CmdErr(c, "%w", &exitError{code: exitCode})
	}
	return waitErr
}
