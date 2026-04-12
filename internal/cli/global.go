// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cli

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	logpkg "github.com/i-zaitsev/dwoe/internal/log"
)

const Prog = "dwoe"

// GlobalFlags define parameters generic for all subcommands.
// The flags configure logging, the location of workspace metadata, and the default proxy policy.
type GlobalFlags struct {
	dataDir   string        // dataDir specifies the directory with workspaces
	logFile   string        // logFile redirects logs to a file
	logLevel  logLevelParam // logLevel specifies the logging level
	logFormat logpkg.Format // logFormat specifies the log output format: JSON or text
	sourceDir string        // sourceDir overrides source.local_path when not set in a task
	model     string        // model overrides agent.model when not set in a task
	taskName  string        // taskName overrides the auto-generated workspace name
	noProxy   bool          // noProxy disables the proxy container
}

// parseGlobalFlags is executed before the subcommand is parsed.
func parseGlobalFlags(args []string) (*GlobalFlags, []string, error) {
	var flags GlobalFlags
	flags.logLevel = logLevelParam(slog.LevelInfo)
	fs := flag.NewFlagSet(Prog, flag.ContinueOnError)

	fs.Usage = func() { /* do not print anything by default */ }

	fs.StringVar(&flags.dataDir, "datadir", defaultDataDir(), "directory with workspace metadata")
	fs.StringVar(&flags.logFile, "logfile", "", "write JSON logs to file")
	fs.Var(&flags.logLevel, "loglevel", "log level (debug, info, warn, error)")
	fs.Var(&flags.logFormat, "logfmt", "log output format (JSON or text)")
	fs.StringVar(&flags.sourceDir, "sourcedir", "", "default source directory for tasks")
	fs.StringVar(&flags.model, "model", "", "default model for tasks")
	fs.StringVar(&flags.taskName, "taskname", "", "override workspace name")
	fs.BoolVar(&flags.noProxy, "noproxy", false, "disable proxy container")

	var help bool
	fs.BoolVar(&help, "h", false, "show help")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	if help {
		return nil, nil, flag.ErrHelp
	}

	return &flags, fs.Args(), nil
}

// defaultDataDir defined as the user's home dir and the program name.
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "." + Prog
	}
	return filepath.Join(home, "."+Prog)
}

// logLevelParam is a flag.Value that parses a log level.
// The wrapper is used to expose the log level to the CLI.
type logLevelParam slog.Level

// Set implements the flag.Value.
func (l *logLevelParam) Set(value string) error {
	var level slog.Level
	if value == "" {
		value = "info"
	}
	switch strings.ToLower(value) {
	case "dbg":
		value = "debug"
	case "err":
		value = "error"
	case "inf":
		value = "info"
	case "wrn":
		value = "warn"
	}
	if err := level.UnmarshalText([]byte(value)); err != nil {
		level = slog.LevelDebug
	}
	*l = logLevelParam(level)
	return nil
}

// String implements the flag.Value.
func (l *logLevelParam) String() string {
	return l.level().String()
}

// level unwraps the log level from CLI option into internal representation.
func (l *logLevelParam) level() slog.Level {
	return slog.Level(*l)
}
