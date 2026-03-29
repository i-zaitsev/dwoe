// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/version"
)

// cmdVersion prints the program version.
type cmdVersion struct{}

// Name returns the subcommand name.
func (c *cmdVersion) Name() string { return "version" }
func (c *cmdVersion) Desc() string { return "Show version" }
func (c *cmdVersion) Args() string { return "" }

// Parse is a no-op; the version command takes no arguments.
func (c *cmdVersion) Parse(_ []string) error {
	return nil
}

// Run prints the current version string to stdout.
func (c *cmdVersion) Run(e *cli.Env) error {
	e.Print("%s\n", version.Get())
	return nil
}
