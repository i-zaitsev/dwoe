// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cli

import "fmt"

// CmdErr is a helper function to wrap a subcommand-specific error into a generic error format.
func CmdErr(cmd Command, format string, args ...any) error {
	return fmt.Errorf("%s: "+format, append([]any{cmd.Name()}, args...)...)
}

// ArgMissingError is returned if a subcommand didn't get the required argument.
type ArgMissingError struct {
	Name string
}

func (e *ArgMissingError) Error() string {
	return e.Name + " is required"
}

// ArgInvalidError is returned if a subcommand was incorrectly configured.
type ArgInvalidError struct {
	Name  string
	Value string
}

func (e *ArgInvalidError) Error() string {
	return "invalid " + e.Name + ": " + e.Value
}

// FlagParseError wraps the standard flag parsing error for type-checking purposes.
type FlagParseError struct {
	err error
}

func (e *FlagParseError) Error() string { return e.err.Error() }
func (e *FlagParseError) Unwrap() error { return e.err }
