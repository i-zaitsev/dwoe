// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package log configures the global [log/slog] logger.
package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Format selects the log output format.
type Format int

const (
	FormatJSON Format = iota
	FormatText
)

func (f *Format) Set(value string) error {
	if value == "" {
		value = "json"
	}
	switch value {
	case "json":
		*f = FormatJSON
	case "text":
		*f = FormatText
	default:
		return fmt.Errorf("invalid logging format: %s", value)
	}
	return nil
}

func (f *Format) String() string {
	switch *f {
	case FormatJSON:
		return "json"
	case FormatText:
		return "text"
	}
	return ""
}

// Opts configures the global logger.
type Opts struct {
	Level      slog.Level
	Format     Format
	Writer     io.Writer
	SourceRoot string
}

// DefaultOpts returns Opts with info-level JSON logging to stderr.
func DefaultOpts() *Opts {
	return &Opts{
		Level:  slog.LevelInfo,
		Format: FormatJSON,
		Writer: os.Stderr,
	}
}

// Setup configures the global [slog] default logger.
func Setup(opts *Opts) {
	if opts == nil {
		opts = DefaultOpts()
	}
	w := opts.Writer
	if w == nil {
		w = os.Stderr
	}
	hOpts := &slog.HandlerOptions{
		Level:       opts.Level,
		AddSource:   true,
		ReplaceAttr: useRelativeSourcePath(opts.SourceRoot),
	}
	var h slog.Handler
	switch opts.Format {
	case FormatText:
		h = slog.NewTextHandler(w, hOpts)
	default:
		h = slog.NewJSONHandler(w, hOpts)
	}
	slog.SetDefault(slog.New(h))
}

// SetupDefault configures the logger with default options.
func SetupDefault() {
	Setup(DefaultOpts())
}

// SetupJSON configures JSON logging at the given level.
func SetupJSON(level slog.Level) {
	Setup(&Opts{Level: level, Format: FormatJSON, Writer: os.Stderr})
}

// SetupVerboseText configures debug-level text logging to stderr.
func SetupVerboseText() {
	Setup(&Opts{Level: slog.LevelDebug, Format: FormatText, Writer: os.Stderr})
}

// useRelativeSourcePath used to replace source attr in slog.HandlerOptions for relative path logging.
func useRelativeSourcePath(sourceRoot string) func(groups []string, a slog.Attr) slog.Attr {
	return func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			if s, ok := a.Value.Any().(*slog.Source); ok {
				s.File = strings.TrimPrefix(s.File, sourceRoot)
			}
		}
		return a
	}
}
