// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package commands implements all CLI subcommands for the dwoe tool.
package commands

import (
	"github.com/i-zaitsev/dwoe/internal/cli"
)

// Registry returns a map of all subcommands.
// The subcommands should be registered via cli.RegisterCommands.
// This allows decoupling the CLI entry point from specific subcommands.
func Registry() map[string]cli.Command {
	subcommands := []cli.Command{
		newCmdBatch(),
		newCmdCollect(),
		newCmdFire(),
		newCmdPatches(),
		new(cmdCreate),
		new(cmdInspect),
		new(cmdRun),
		new(cmdStart),
		new(cmdStop),
		new(cmdDestroy),
		new(cmdList),
		new(cmdLogs),
		new(cmdStatus),
		new(cmdVersion),
		new(cmdWeb),
	}

	registry := make(map[string]cli.Command, len(subcommands))

	for _, cmd := range subcommands {
		registry[cmd.Name()] = cmd
	}

	return registry
}
