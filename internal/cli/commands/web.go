// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"flag"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/web"
)

// cmdWeb starts the web dashboard.
//
// The dashboard provides a browser-based UI for monitoring workspaces.
// The addr flag controls the listen address (default ":8080").
type cmdWeb struct {
	addr string
}

// Name returns the subcommand name.
func (c *cmdWeb) Name() string { return "web" }
func (c *cmdWeb) Desc() string { return "Start web dashboard" }
func (c *cmdWeb) Args() string { return "[--addr ADDR]" }

// Parse expects the arguments for the subcommand.
// The optional addr flag sets the HTTP listen address.
func (c *cmdWeb) Parse(args []string) error {
	_, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.StringVar(&c.addr, "addr", "127.0.0.1:8080", "listen address")
	})
	return err
}

// Run starts the web server and blocks until it returns.
func (c *cmdWeb) Run(e *cli.Env) error {
	manager, err := e.Manager()

	if err != nil {
		return err
	}

	srv := web.NewServer(c.addr)
	web.Routes(srv)
	srv.SetSource(manager)

	e.Print("Dashboard: http://%s\n", c.addr)

	return srv.ListenAndServe(e.Context())
}
