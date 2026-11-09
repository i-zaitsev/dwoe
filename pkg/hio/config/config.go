// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package config holds the settings of the dashboard server.
package config

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Config is the set of settings of the server.
// The zero value listens on all interfaces at port 0 and logs at info level.
type Config struct {
	Addr     string
	Port     int
	DataDir  string
	UIDir    string // read templates and static files from here instead of the embedded ones
	LogLevel slog.Level
}

// logLevelFlag lets a flag set a slog.Level.
type logLevelFlag struct {
	level *slog.Level
}

func (f logLevelFlag) String() string {
	if f.level == nil {
		return ""
	}
	return f.level.String()
}

func (f logLevelFlag) Set(s string) error {
	switch s {
	case "debug":
		*f.level = slog.LevelDebug
	case "info":
		*f.level = slog.LevelInfo
	case "warn":
		*f.level = slog.LevelWarn
	case "error":
		*f.level = slog.LevelError
	default:
		return fmt.Errorf("invalid log level: %s", s)
	}
	return nil
}

func newFlagSet(cfg *Config) *flag.FlagSet {
	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Addr, "addr", "127.0.0.1", "server address")
	fs.IntVar(&cfg.Port, "port", 8080, "server port")
	fs.StringVar(&cfg.DataDir, "datadir", defaultDataDir(), "data directory")
	fs.StringVar(&cfg.UIDir, "ui-dir", "", "read templates and static files from this directory instead of the embedded ones")
	fs.Var(logLevelFlag{level: &cfg.LogLevel}, "log-level", "log level (debug|info|warn|error)")

	return fs
}

// Usage writes the flags with their defaults to w.
func Usage(w io.Writer) {
	fs := newFlagSet(&Config{LogLevel: slog.LevelInfo})
	fs.SetOutput(w)
	_, _ = fmt.Fprintln(w, "Usage of web:")
	fs.PrintDefaults()
}

// Parse reads the settings from command-line arguments.
// For -h and -help the error wraps flag.ErrHelp.
func Parse(args []string) (Config, error) {
	cfg := Config{LogLevel: slog.LevelInfo}

	fs := newFlagSet(&cfg)
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}

	return cfg, nil
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".dwoe"
	}
	return filepath.Join(home, ".dwoe")
}
