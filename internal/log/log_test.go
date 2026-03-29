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
)

func TestSetup_DefaultLevel(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	Setup(&Opts{Level: slog.LevelInfo, Format: FormatText, Writer: &buf})

	slog.Info("visible")
	slog.Debug("hidden")

	out := buf.String()
	if !strings.Contains(out, "visible") {
		t.Error("expected Info message in output")
	}
	if strings.Contains(out, "hidden") {
		t.Error("Debug message should not appear at Info level")
	}
	if !strings.Contains(out, "source=") {
		t.Error("expected source location in output")
	}
}

func TestSetup_VerboseLevel(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	Setup(&Opts{Level: slog.LevelDebug, Format: FormatText, Writer: &buf})

	slog.Debug("shown")

	if !strings.Contains(buf.String(), "shown") {
		t.Error("expected Debug message in verbose mode")
	}
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
	if m["msg"] != "test: hello" {
		t.Errorf("msg = %q, want %q", m["msg"], "test: hello")
	}
	if m["key"] != "value" {
		t.Errorf("key = %q, want %q", m["key"], "value")
	}
	if m["source"] == nil {
		t.Error("expected source field in JSON output")
	}
}

func TestDefaultOpts(t *testing.T) {
	opts := DefaultOpts()
	if opts.Level != slog.LevelInfo {
		t.Errorf("Level = %v, want %v", opts.Level, slog.LevelInfo)
	}
	if opts.Format != FormatJSON {
		t.Errorf("Format = %v, want %v", opts.Format, FormatJSON)
	}
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
		if got := tt.f.String(); got != tt.want {
			t.Errorf("Format(%d).String() = %q, want %q", tt.f, got, tt.want)
		}
	}
}
