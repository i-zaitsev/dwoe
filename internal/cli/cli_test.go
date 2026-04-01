// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/i-zaitsev/dwoe/internal/assert"
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
		{"noproxy", []string{"--noproxy", "run", "t.yaml"}, defaultDataDir(), "", "", true, []string{"run", "t.yaml"}},
		{"sourcedir", []string{"--sourcedir", "/src", "batch", "tasks"}, defaultDataDir(), "/src", "", false, []string{"batch", "tasks"}},
		{"model", []string{"--model", "claude-sonnet-4-6", "run", "t.yaml"}, defaultDataDir(), "", "claude-sonnet-4-6", false, []string{"run", "t.yaml"}},
		{"sourcedir_and_model", []string{"--sourcedir", "/code", "--model", "m", "batch", "."}, defaultDataDir(), "/code", "m", false, []string{"batch", "."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, rest, err := parseGlobalFlags(tt.args)

			assert.NotErr(t, err)
			assert.Equal(t, flags.dataDir, tt.dataDir)
			assert.Equal(t, flags.sourceDir, tt.sourceDir)
			assert.Equal(t, flags.model, tt.model)
			assert.Equal(t, flags.noProxy, tt.noProxy)

			if diff := cmp.Diff(tt.rest, rest); diff != "" {
				t.Fatalf("rest mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestRunNoArgs(t *testing.T) {
	e, stdout, _ := testEnv()

	err := Run(e, nil)

	assert.NotErr(t, err)
	assert.Contains(t, stdout.String(), "Usage:")
}

func TestRunUnknownCommand(t *testing.T) {
	e, _, _ := testEnv()

	err := Run(e, []string{"bogus"})

	assert.Err(t, err)
	assert.ErrIs(t, err, ErrUnknownCommand)
}

func TestRunHelp(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			e, stdout, _ := testEnv()
			err := Run(e, []string{arg})
			assert.NotErr(t, err)

			out := stdout.String()
			assert.ContainsAll(t, out, "Usage:", "Flags:", "Commands:")
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
		assert.Contains(t, out, want)
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
	assert.NotErr(t, err)
	out := stdout.String()
	assert.Contains(t, out, "A test command")
	assert.Contains(t, out, "Usage:")
}
