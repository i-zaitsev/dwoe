// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/i-zaitsev/dwoe/internal/assert"
)

func init() {
	none := make(map[string]Command)
	RegisterCommands(none)
}

func testEnv() (*Env, *bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer
	return NewEnv(&stdout, &stderr), &stdout, &stderr
}

func TestParseGlobalFlags_FromArgs(t *testing.T) {
	dataDir := defaultDataDir()

	testCases := []struct {
		args []string
		want GlobalFlags
	}{
		{
			args: nil,
			want: GlobalFlags{dataDir: dataDir},
		},
		{
			args: []string{"list"},
			want: GlobalFlags{dataDir: dataDir},
		},
		{
			args: []string{"--datadir", "/tmp", "run", "task.yaml"},
			want: GlobalFlags{dataDir: "/tmp"},
		},
		{
			args: []string{"--noproxy", "run", "task.yaml"},
			want: GlobalFlags{dataDir: dataDir, noProxy: true},
		},
		{
			args: []string{"--sourcedir", "/src", "batch", "tasks"},
			want: GlobalFlags{dataDir: dataDir, sourceDir: "/src"},
		},
		{
			args: []string{"--model", "claude-sonnet-4-6", "run", "task.yaml"},
			want: GlobalFlags{dataDir: dataDir, model: "claude-sonnet-4-6"},
		},
		{
			args: []string{"--taskname", "my-task", "fire", "--repo", "https://example.com/repo"},
			want: GlobalFlags{dataDir: dataDir, taskName: "my-task"},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("args=%v", tc.args), func(t *testing.T) {
			got, _, err := parseGlobalFlags(tc.args)
			assert.NotErr(t, err)
			if diff := cmp.Diff(tc.want, *got, cmp.AllowUnexported(GlobalFlags{})); diff != "" {
				t.Fatalf("flags mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseGlobalFlags_RestArgs(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		args, rest []string
	}{
		{nil, nil},
		{[]string{"--datadir", "/tmp"}, []string{}},
		{[]string{"--noproxy", "run", "task.yaml"}, []string{"run", "task.yaml"}},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("args=%v", tc.args), func(t *testing.T) {
			_, rest, err := parseGlobalFlags(tc.args)
			assert.NotErr(t, err)
			if diff := cmp.Diff(tc.rest, rest); diff != "" {
				t.Fatalf("rest mismatch (-want +got):\n%s", diff)
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
