// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package log

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestSetup_DefaultLevel(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	Setup(&Opts{Level: slog.LevelInfo, Format: FormatText, Writer: &buf})

	slog.Info("visible")
	slog.Debug("hidden")

	out := buf.String()
	assert.Contains(t, out, "visible")
	if strings.Contains(out, "hidden") {
		t.Error("Debug message should not appear at Info level")
	}
	assert.Contains(t, out, "source=")
}

func TestSetup_VerboseLevel(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	Setup(&Opts{Level: slog.LevelDebug, Format: FormatText, Writer: &buf})

	slog.Debug("shown")

	assert.Contains(t, buf.String(), "shown")
}

func TestSetup_JSONFormat(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	Setup(&Opts{Level: slog.LevelInfo, Format: FormatJSON, Writer: &buf})

	slog.Info("test: hello", "key", "value")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	assert.Equal(t, m["msg"], "test: hello")
	assert.Equal(t, m["key"], "value")
	if m["source"] == nil {
		t.Error("expected source field in JSON output")
	}
}

func TestDefaultOpts(t *testing.T) {
	opts := DefaultOpts()
	assert.Equal(t, opts.Level, slog.LevelInfo)
	assert.Equal(t, opts.Format, FormatJSON)
	if opts.Writer == nil {
		t.Error("Writer should not be nil")
	}
}

func TestSetup_NilOpts(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	Setup(nil) // should not panic
}

func TestFormat_FlagValue(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    Format
		wantErr bool
	}{
		{"json", []string{"-format", "json"}, FormatJSON, false},
		{"text", []string{"-format", "text"}, FormatText, false},
		{"invalid", []string{"-format", "invalid"}, FormatJSON, true},
		{"empty", []string{"-format", ""}, FormatJSON, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f Format
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			fs.Var(&f, "format", "usage")
			err := fs.Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && f != tt.want {
				t.Errorf("f = %v, want %v", f, tt.want)
			}
		})
	}
}

func TestFormat_String(t *testing.T) {
	tests := []struct {
		f    Format
		want string
	}{
		{FormatJSON, "json"},
		{FormatText, "text"},
		{Format(99), ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.f.String(), tt.want)
	}
}
