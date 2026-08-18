// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	ansiReset  = "\033[0m"
	ansiDim    = "\033[2m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
)

// Configurable helper that returns a runtime frame for source location logging.
// Helpful for overriding in tests.
var getFrame = func(pc uintptr) runtime.Frame {
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	return frame
}

// PrettyHandler renders records in a readable format, one line each:
//
//	[2026-08-17 20:23:09][INF] cli: run taskPath=task.yaml detach=false
//	[2026-08-17 20:23:39][ERR] wait: container err="context canceled" (internal/docker/client.go:192)
//
// The message starts at a fixed column. The source location is appended only to warnings and errors.
// Colors are used when the writer is a terminal
type PrettyHandler struct {
	w     io.Writer
	mu    *sync.Mutex   // guards buf and writing to w
	buf   *bytes.Buffer // the inner handler renders attributes here
	inner slog.Handler
	root  string
	color bool
}

// NewPrettyHandler creates an instance of PrettyHandler for readable logging.
// The level defines logging level, and root is used for relative source path resolution
// based on the path to the project's root.
func NewPrettyHandler(w io.Writer, level slog.Level, root string) *PrettyHandler {
	buf := &bytes.Buffer{}
	return &PrettyHandler{
		w:   w,
		mu:  &sync.Mutex{},
		buf: buf,
		inner: slog.NewTextHandler(buf, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: dropBuiltinAttrs,
		}),
		root:  root,
		color: isTerminal(w),
	}
}

// Enabled checks if the logging level is active.
func (h *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// WithAttrs returns a new handler with the given attributes.
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	c := *h
	c.inner = h.inner.WithAttrs(attrs)
	return &c
}

// WithGroup returns a new handler with the given group name.
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	c := *h
	c.inner = h.inner.WithGroup(name)
	return &c
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	// The buffer is shared with handlers derived by WithAttrs and WithGroup.
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buf.Reset()
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}
	attrs := strings.TrimRight(h.buf.String(), "\n")

	var b strings.Builder
	b.WriteByte('[')
	b.WriteString(r.Time.Format(time.DateTime))
	b.WriteString("][")
	h.writeLevel(&b, r.Level)
	b.WriteString("] ")
	b.WriteString(r.Message)

	if attrs != "" {
		b.WriteByte(' ')
		b.WriteString(attrs)
	}
	if r.Level >= slog.LevelWarn {
		if src := h.source(r.PC); src != "" {
			b.WriteByte(' ')
			h.writeColored(&b, ansiDim, "("+src+")")
		}
	}
	b.WriteByte('\n')

	_, err := io.WriteString(h.w, b.String())
	return err
}

// writeLevel replaces logging levels with short aliases and colors it for visibility when
// rendered to interactive terminal.
func (h *PrettyHandler) writeLevel(b *strings.Builder, level slog.Level) {
	tag, color := "ERR", ansiRed
	switch {
	case level < slog.LevelInfo:
		tag, color = "DBG", ansiDim
	case level < slog.LevelWarn:
		tag, color = "INF", ansiGreen
	case level < slog.LevelError:
		tag, color = "WRN", ansiYellow
	}
	h.writeColored(b, color, tag)
}

// writeColored uses control sequence to render colored text for interactive devices.
func (h *PrettyHandler) writeColored(b *strings.Builder, color, text string) {
	if !h.color {
		b.WriteString(text)
		return
	}
	b.WriteString(color)
	b.WriteString(text)
	b.WriteString(ansiReset)
}

// source formats the call site as a path relative to the module root.
func (h *PrettyHandler) source(pc uintptr) string {
	if pc == 0 {
		return ""
	}
	frame := getFrame(pc)
	if frame.File == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", strings.TrimPrefix(frame.File, h.root), frame.Line)
}

// isTerminal reports whether io.Writer is a character device.
// Color codes are left out of files and pipes.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// dropBuiltinAttrs removes the fields rendered by PrettyHandler.
// It leaves the inner handler to write only the caller's attributes.
// Nested keys are kept so that attributes inside a group still appear.
func dropBuiltinAttrs(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey, slog.LevelKey, slog.MessageKey:
		return slog.Attr{}
	}
	return a
}
