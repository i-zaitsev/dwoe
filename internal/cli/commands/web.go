// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"errors"
	"flag"
	"log/slog"
	"net"
	"strconv"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/pkg/hio/config"
	"github.com/i-zaitsev/dwoe/web"
	"github.com/i-zaitsev/pkg/cmd"
)

// cmdWeb starts the web dashboard.
//
// The dashboard provides a browser-based UI for monitoring workspaces.
// It listens on 127.0.0.1:8080 by default. The host and port flags change one
// part of the address, the addr flag sets both as host:port and excludes the
// other two. The ui-dir flag reads templates and static files from a directory
// instead of the embedded ones.
type cmdWeb struct {
	host  string
	port  int
	uiDir string
}

// Name returns the subcommand name.
func (c *cmdWeb) Name() string { return "web" }
func (c *cmdWeb) Desc() string { return "Start web dashboard" }
func (c *cmdWeb) Args() string {
	return "[--addr HOST:PORT | --host HOST --port PORT] [--ui-dir DIR]"
}

// Parse expects the arguments for the subcommand.
// The listen address comes from addr when given, else from host and port.
func (c *cmdWeb) Parse(args []string) error {
	var addr string
	fs := cmd.NewFlagSet(c)
	fs.StringVar(&addr, "addr", "", "listen address as host:port")
	fs.StringVar(&c.host, "host", "127.0.0.1", "listen host")
	fs.IntVar(&c.port, "port", 8080, "listen port")
	fs.StringVar(&c.uiDir, "ui-dir", "", "read templates and static files from this directory")
	err := fs.Parse(args)
	if err != nil {
		return cmd.Errorf(c, "%w", err)
	}
	if addr == "" {
		return nil
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "host" || f.Name == "port" {
			err = errors.New("--addr cannot be used with --host or --port")
		}
	})
	if err != nil {
		return cmd.Errorf(c, "%w", err)
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return cmd.Errorf(c, "--addr: %w", err)
	}
	c.port, err = strconv.Atoi(port)
	if err != nil {
		return cmd.Errorf(c, "--addr: invalid port %q", port)
	}
	c.host = host
	return nil
}

// Run starts the web server and blocks until it stops.
func (c *cmdWeb) Run(e *cli.Env) error {
	manager, err := e.Manager()
	if err != nil {
		return err
	}

	env := config.NewEnv(config.Config{
		Addr:     c.host,
		Port:     c.port,
		UIDir:    c.uiDir,
		LogLevel: slog.LevelWarn,
	})

	e.Print("Dashboard: http://%s\n", env.ServerAddr())

	return web.Serve(e.Context(), env, web.NewModels(manager))
}
