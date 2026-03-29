// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cli

import (
	"bufio"
	"context"
	"flag"
	"io"
	"log/slog"
	"strings"
	"time"
)

// ParseFlags is a generic flag-parsing util for subcommands.
// The register function is a callback that configures a flag set required by command.
// The parser has output disabled and empty usage which is delegated to the top-level entry point.
func ParseFlags(cmd Command, args []string, register func(*flag.FlagSet)) (*flag.FlagSet, error) {
	name := cmd.Name()
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {}
	fs.SetOutput(io.Discard)
	if register != nil {
		register(fs)
	}
	if err := fs.Parse(args); err != nil {
		return nil, CmdErr(cmd, "%w", &FlagParseError{err: err})
	}
	return fs, nil
}

// ScanLogs reads logs from a worker.
// The function expects a generic io.ReadCloser and sends the collected logs
// to the lines channel. The internal buffer in bufio.Scanner is limited to 1 mb.
func ScanLogs(ctx context.Context, logs io.ReadCloser, lines chan<- string) {
	const (
		kb       = 1024
		mb       = 1024 * kb
		sentinel = "<promise>DONE</promise>"
	)

	defer close(lines)
	defer logs.Close()

	// AfterFunc closes the reader when the context is cancelled (e.g., SIGINT),
	// unblocking the scanner. stop() cancels the callback if the scanner exits
	// normally first, preventing a concurrent double-close with defer logs.Close().
	stop := context.AfterFunc(ctx, func() { logs.Close() })
	defer func() {
		if stopped := stop(); !stopped {
			slog.Debug("run: scanner: closed by context cancellation")
		}
	}()

	scanner := bufio.NewScanner(logs)
	scanner.Buffer(make([]byte, 0, 64*kb), mb)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, sentinel) {
			break
		}
		lines <- line
	}

	if err := scanner.Err(); err != nil {
		slog.Warn("run: scanner", "err", err)
	}
}

// CutIfLong cuts the string to a predefined maximum size.
// Used to truncate long hashes and IDs for display purposes.
func CutIfLong(s string) string {
	const maxSize = 8
	if len(s) > maxSize {
		return s[:maxSize]
	}
	return s
}

// FmtTime formats a time.Time to a human-readable string.
func FmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateTime)
}
