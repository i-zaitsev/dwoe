// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/cli/commands"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	cli.RegisterCommands(commands.Registry())
	env := cli.NewEnv(os.Stdout, os.Stderr)
	env.SetContext(ctx)

	if err := cli.Run(env, os.Args[1:]); err != nil {
		env.Error("error: %v\n", err)
		return 1
	}

	return 0
}
