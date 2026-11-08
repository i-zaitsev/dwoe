// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/pkg/hio/config"
	"github.com/i-zaitsev/dwoe/web"
	"github.com/i-zaitsev/pkg/cmd"
)

// cmdWeb starts the web dashboard.
//
// The dashboard provides a browser-based UI for monitoring workspaces.
// The dashboard listens on the loopback address; the port flag changes the
// port (default 8080). The ui-dir flag reads templates and static files from
// a directory instead of the embedded ones.
type cmdWeb struct {
	port  int
	uiDir string
}

// Name returns the subcommand name.
func (c *cmdWeb) Name() string { return "web" }
func (c *cmdWeb) Desc() string { return "Start web dashboard" }
func (c *cmdWeb) Args() string { return "[--port PORT] [--ui-dir DIR]" }

// Parse expects the arguments for the subcommand.
// The optional port flag sets the HTTP listen port.
func (c *cmdWeb) Parse(args []string) error {
	fs := cmd.NewFlagSet(c)
	fs.IntVar(&c.port, "port", 8080, "listen port")
	fs.StringVar(&c.uiDir, "ui-dir", "", "read templates and static files from this directory")
	err := fs.Parse(args)
	if err != nil {
		return cmd.Errorf(c, "%w", err)
	}
	return nil
}

// Run starts the web server and blocks until it stops.
func (c *cmdWeb) Run(e *cli.Env) error {
	manager, err := e.Manager()
	if err != nil {
		return err
	}

	env := config.NewEnv(config.Config{
		Addr:     "127.0.0.1",
		Port:     c.port,
		UIDir:    c.uiDir,
		LogLevel: slog.LevelWarn,
	})

	e.Print("Dashboard: http://%s\n", env.ServerAddr())

	return web.Serve(e.Context(), env, web.NewModels(manager))
}
