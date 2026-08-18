// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"fmt"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// loadTaskConfig reads the global config from dataDir and uses it to fill the gaps
// in the task configuration read from taskPath.
func loadTaskConfig(taskPath, dataDir string) (*config.Task, error) {
	global, err := config.LoadGlobalConfig(dataDir)
	if err != nil {
		return nil, fmt.Errorf("load global config: %w", err)
	}
	return config.LoadTaskConfig(taskPath, global)
}

// resolveWorkspace creates a manager from the env and resolves the named workspace,
// returning an error if either step fails.
func resolveWorkspace(cmd cli.Command, e *cli.Env, nameOrID string) (*workspace.Manager, *workspace.Workspace, error) {
	manager, err := e.Manager()
	if err != nil {
		return nil, nil, cli.CmdErr(cmd, "%w", err)
	}
	ws, err := manager.Resolve(nameOrID)
	if err != nil {
		return nil, nil, cli.CmdErr(cmd, "%w", err)
	}
	return manager, ws, nil
}

// resolveCompletedWorkspace creates a manager from the env and resolves a completed workspace.
// It mirrors resolveWorkspace, but the resolved workspace must have a sentinel file.
func resolveCompletedWorkspace(cmd cli.Command, e *cli.Env, nameOrID string) (*workspace.Workspace, error) {
	manager, err := e.Manager()
	if err != nil {
		return nil, cli.CmdErr(cmd, "%w", err)
	}
	ws, err := manager.ResolveCompleted(nameOrID)
	if err != nil {
		return nil, cli.CmdErr(cmd, "%w", err)
	}
	return ws, nil
}

// applyTaskOverrides applies global env overrides (no-proxy and task name) to taskCfg,
// preferring name over the env task name.
func applyTaskOverrides(e *cli.Env, taskCfg *config.Task, name string) {
	if e.NoProxy() {
		taskCfg.NoProxy = true
	}
	if name != "" {
		taskCfg.Name = name
	} else if e.TaskName() != "" {
		taskCfg.Name = e.TaskName()
	}
}
