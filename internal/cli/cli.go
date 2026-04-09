// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package cli implements the command-line interface for the dwoe tool.
package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/i-zaitsev/dwoe/internal/config"
	logpkg "github.com/i-zaitsev/dwoe/internal/log"
	"github.com/i-zaitsev/dwoe/internal/version"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// ErrUnknownCommand is returned when a subcommand name is not in the registry.
var ErrUnknownCommand = errors.New("unknown command")

// registry stores a mapping from a subcommand name to its implementation.
// The subcommands are registered dynamically to decouple the main entry point
// from particular command implementations.
var registry map[string]Command

// Command implements a certain part of the CLI functionality.
//
// Name returns the user-visible name of the command.
// Parse takes a list of args and takes the relevant parameters from there.
// Run executes the command.
//
// Most of the time commands trigger execution of external services like
// building a docker image or running a container. In other cases, it
// pulls information about workspaces from the configured location.
type Command interface {
	Name() string
	Desc() string
	Args() string
	Parse(args []string) error
	Run(e *Env) error
}

// RegisterCommands defines a list of subcommands available via the binary.
// The commands are saved to the module-level map.
func RegisterCommands(r map[string]Command) {
	registry = r
}

// Run executes a command in the given Env.
// The Env decouples command execution from stdio, context and workspace manager.
// These parameters depend on the requirements and can be configured before execution.
func Run(env *Env, args []string) error {
	global, rest, err := parseGlobalFlags(args)
	if errors.Is(err, flag.ErrHelp) {
		printUsage(env)
		return nil
	}
	if err != nil {
		return err
	}

	opts := logpkg.DefaultOpts()
	opts.Level = global.logLevel.level()
	opts.Format = global.logFormat
	opts.SourceRoot = getSourceRoot()

	if global.logFile == "" {
		opts.Writer = env.stderr
	} else {
		f, errOpen := os.OpenFile(global.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if errOpen != nil {
			return fmt.Errorf("cannot open log file: %w", errOpen)
		}
		defer func() {
			_ = f.Close()
		}()
		opts.Writer = f
	}

	logpkg.Setup(opts)

	slog.Debug("cli: init", "datadir", global.dataDir)

	env.dataDir = global.dataDir
	env.noProxy = global.noProxy
	env.model = global.model

	configPath, errInit := config.InitConfig(env.dataDir)

	if opts.Level == slog.LevelDebug {
		switch {
		case errors.Is(errInit, config.ErrConfigExists):
			env.Print("Reading global config: %s\n", configPath)
		case errInit != nil:
			slog.Warn("cli: init config", "err", errInit)
		default:
			env.Print("Created config: %s\n", configPath)
		}
	}

	if global.sourceDir != "" {
		abs, errAbs := filepath.Abs(global.sourceDir)
		if errAbs != nil {
			return fmt.Errorf("resolve --sourcedir: %w", errAbs)
		}
		env.sourceDir = abs
	}

	env.newManager = func() (*workspace.Manager, error) {
		// creates a new workspace manager in the provided directory
		return workspace.NewManager(global.dataDir)
	}

	if len(rest) == 0 {
		printUsage(env)
		return nil
	}

	cmd, cmdArgs := rest[0], rest[1:]

	// multiple variants printing help message
	helpAlias := map[string]bool{
		"help":   true,
		"-h":     true,
		"-help":  true,
		"--help": true,
	}
	if helpAlias[cmd] {
		printUsage(env)
		return nil
	}

	err = dispatchCmd(env, cmd, cmdArgs)
	if errors.Is(err, ErrUnknownCommand) {
		printUsage(env)
	}

	return err
}

// dispatchCmd invokes the selected subcommand.
// The dispatch looks up the subcommand in the registry and calls Command interface
// methods to parse the args and run the configured subcommand instance.
func dispatchCmd(env *Env, cmdName string, cmdArgs []string) error {
	cmd, ok := registry[cmdName]

	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownCommand, cmdName)
	}

	slog.Debug("cli: dispatch", "command", cmdName)

	if err := cmd.Parse(cmdArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printCmdUsage(env, cmd)
			return nil
		}
		slog.Error("cli: dispatch failed", "err", err)
		return err
	}

	return cmd.Run(env)
}

// printUsage prints the usage string.
func printUsage(env *Env) {
	env.Print("%s", buildUsage())
}

// buildUsage dynamically creates a string describing the usage of the tool.
// Each subcommand provides the necessary information to build the message.
// Global flags are defined directly.
func buildUsage() string {
	var buf bytes.Buffer

	_, _ = fmt.Fprintf(&buf, "%s %s\n\nUsage:\n\t%s [flags] <command> [args]\n\n", Prog, version.Get(), Prog)
	buf.WriteString("Flags:\n")
	buf.WriteString("\t--datadir <dir>     Data directory (default: ~/.dwoe)\n")
	buf.WriteString("\t--logfile <path>    Write JSON logs to file\n")
	buf.WriteString("\t--loglevel <level>  Log level: debug, info, warn, error (default: warn)\n")
	buf.WriteString("\t--logfmt <format>   Log format: text, json (default: json)\n")
	buf.WriteString("\t--sourcedir <path>  Default source directory for tasks\n")
	buf.WriteString("\t--model <model>     Default model for tasks\n")
	buf.WriteString("\t--noproxy           Disable proxy container\n")
	buf.WriteString("\nCommands:\n")

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}

	sort.Strings(names)

	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)
	for _, name := range names {
		cmd := registry[name]
		args := cmd.Args()
		if args != "" {
			_, _ = fmt.Fprintf(tw, "\t%s\t%s\t%s\n", name, args, cmd.Desc())
		} else {
			_, _ = fmt.Fprintf(tw, "\t%s\t\t%s\n", name, cmd.Desc())
		}
	}

	_ = tw.Flush()

	return buf.String()
}

// printCmdUsage prints the usage of a subcommand depending on its requirements.
func printCmdUsage(env *Env, cmd Command) {
	if args := cmd.Args(); args != "" {
		env.Print("Usage: %s %s %s\n\n", Prog, cmd.Name(), args)
	} else {
		env.Print("Usage: %s %s\n\n", Prog, cmd.Name())
	}
	env.Print("%s\n", cmd.Desc())
}
