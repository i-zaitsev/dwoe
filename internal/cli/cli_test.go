// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

type stubCmd struct {
	name string
	desc string
	args string
}

func (s *stubCmd) Name() string         { return s.name }
func (s *stubCmd) Desc() string         { return s.desc }
func (s *stubCmd) Args() string         { return s.args }
func (s *stubCmd) Parse([]string) error { return nil }
func (s *stubCmd) Run(*Env) error       { return nil }

type helpableStubCmd struct {
	name string
	desc string
	args string
}

func (s *helpableStubCmd) Name() string   { return s.name }
func (s *helpableStubCmd) Desc() string   { return s.desc }
func (s *helpableStubCmd) Args() string   { return s.args }
func (s *helpableStubCmd) Run(*Env) error { return nil }

func (s *helpableStubCmd) Parse(args []string) error {
	_, err := ParseFlags(s, args, nil)
	return err
}

func init() {
	none := make(map[string]Command)
	RegisterCommands(none)
}

func testEnv() (*Env, *bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer
	return NewEnv(&stdout, &stderr), &stdout, &stderr
}

func TestParseGlobalFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		dataDir   string
		sourceDir string
		model     string
		noProxy   bool
		rest      []string
	}{
		{"no args", nil, defaultDataDir(), "", "", false, nil},
		{"command only", []string{"list"}, defaultDataDir(), "", "", false, []string{"list"}},
		{"datadir", []string{"--datadir", "/tmp", "run", "t.yaml"}, "/tmp", "", "", false, []string{"run", "t.yaml"}},
		{"no-proxy", []string{"--no-proxy", "run", "t.yaml"}, defaultDataDir(), "", "", true, []string{"run", "t.yaml"}},
		{"sourceDir", []string{"--sourceDir", "/src", "batch", "tasks"}, defaultDataDir(), "/src", "", false, []string{"batch", "tasks"}},
		{"model", []string{"--model", "claude-sonnet-4-6", "run", "t.yaml"}, defaultDataDir(), "", "claude-sonnet-4-6", false, []string{"run", "t.yaml"}},
		{"sourceDir_and_model", []string{"--sourceDir", "/code", "--model", "m", "batch", "."}, defaultDataDir(), "/code", "m", false, []string{"batch", "."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, rest, err := parseGlobalFlags(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if flags.dataDir != tt.dataDir {
				t.Errorf("dataDir = %q, want %q", flags.dataDir, tt.dataDir)
			}
			if flags.sourceDir != tt.sourceDir {
				t.Errorf("sourceDir = %q, want %q", flags.sourceDir, tt.sourceDir)
			}
			if flags.model != tt.model {
				t.Errorf("model = %q, want %q", flags.model, tt.model)
			}
			if flags.noProxy != tt.noProxy {
				t.Errorf("noProxy = %v, want %v", flags.noProxy, tt.noProxy)
			}
			if diff := cmp.Diff(tt.rest, rest); diff != "" {
				t.Fatalf("rest mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestRunNoArgs(t *testing.T) {
	e, stdout, _ := testEnv()
	if err := Run(e, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Error("expected usage output")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	e, _, _ := testEnv()
	err := Run(e, []string{"bogus"})
	if err == nil {
		t.Fatal("expected error")
	}
	testutil.WantErr(t, err, ErrUnknownCommand)
}

func TestRunHelp(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			e, stdout, _ := testEnv()
			if err := Run(e, []string{arg}); err != nil {
				t.Fatal(err)
			}
			out := stdout.String()
			for _, want := range []string{"Usage:", "Flags:", "Commands:"} {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q", want)
				}
			}
		})
	}
}

func TestBuildUsage(t *testing.T) {
	prev := registry
	defer func() { registry = prev }()

	registry = map[string]Command{
		"alpha": &stubCmd{name: "alpha", desc: "First command", args: "<file>"},
		"beta":  &stubCmd{name: "beta", desc: "Second command", args: ""},
	}

	out := buildUsage()

	for _, want := range []string{"Commands:", "alpha", "beta", "First command", "Second command", "<file>"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}

	alphaIdx := strings.Index(out, "alpha")
	betaIdx := strings.Index(out, "beta")
	if alphaIdx > betaIdx {
		t.Error("commands not sorted alphabetically")
	}
}

func TestDispatchCmdHelp(t *testing.T) {
	prev := registry
	defer func() { registry = prev }()

	registry = map[string]Command{
		"test": &helpableStubCmd{name: "test", desc: "A test command", args: "<arg>"},
	}

	e, stdout, _ := testEnv()
	err := dispatchCmd(e, "test", []string{"-h"})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "A test command") {
		t.Errorf("output missing description, got: %s", out)
	}
	if !strings.Contains(out, "Usage:") {
		t.Errorf("output missing Usage:, got: %s", out)
	}
}
