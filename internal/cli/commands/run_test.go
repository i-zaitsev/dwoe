// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/testfake"
	"github.com/i-zaitsev/dwoe/internal/testutil"
	"github.com/i-zaitsev/dwoe/internal/workspace"
	"github.com/i-zaitsev/pkg/assert"
)

func TestRunCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		args       []string
		wantPath   string
		wantName   string
		wantDetach bool
	}{
		{
			name:     "positional_only",
			args:     []string{"task.yaml"},
			wantPath: "task.yaml",
		},
		{
			name:       "detach_long",
			args:       []string{"--detach", "task.yaml"},
			wantPath:   "task.yaml",
			wantDetach: true,
		},
		{
			name:       "detach_short",
			args:       []string{"-d", "task.yaml"},
			wantPath:   "task.yaml",
			wantDetach: true,
		},
		{
			name:     "with_name",
			args:     []string{"-name", "custom", "task.yaml"},
			wantPath: "task.yaml",
			wantName: "custom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := new(cmdRun)
			assert.ErrVal(t, cmd.Parse(tc.args)).Nil().Fatal()
			assert.Eq(t, cmd.taskPath, tc.wantPath)
			assert.Eq(t, cmd.name, tc.wantName)
			assert.Eq(t, cmd.detach, tc.wantDetach)
		})
	}
}

func TestRunCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdRun))
}

func TestRunCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("detach", func(t *testing.T) {
		t.Parallel()
		ts, taskFile := createTestTask(t, "", 0)
		cmd := &cmdRun{taskPath: taskFile, detach: true}
		assert.ErrVal(t, cmd.Run(ts.env)).Nil().Fatal()
		assert.ContainsAll(t, ts.stdout.String(), "started", "Status: running")
	})

	t.Run("attached_success", func(t *testing.T) {
		t.Parallel()
		ts, taskFile := createTestTask(t, "line1\nline2\n<promise>DONE</promise>\n", 0)
		cmd := &cmdRun{taskPath: taskFile}
		assert.ErrVal(t, cmd.Run(ts.env)).Nil().Fatal()
		assert.ContainsAll(t, ts.stdout.String(), "line1", "line2", "completed")
	})

	t.Run("attached_fail", func(t *testing.T) {
		t.Parallel()
		ts, taskFile := createTestTask(t, "error output\n", 1)
		cmd := &cmdRun{taskPath: taskFile}

		err := cmd.Run(ts.env)

		var exitErr *exitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("err type = %T, want *exitError", err)
		}
		assert.Eq(t, exitErr.code, 1)
		assert.ContainsAll(t, ts.stdout.String(), "failed")
	})
}

func TestRunCmd_Run_Interrupted(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.docker.ContainerLogsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		return logReader("<promise>DONE</promise>\n"), nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	setup.env.SetContext(ctx)

	taskFile := writeTaskFile(t, t.TempDir(), "interrupt-test")
	cmd := &cmdRun{taskPath: taskFile}

	err := cmd.Run(setup.env)

	assert.ErrVal(t, err).Is(errRunInterrupted).Fatal()
	assert.ContainsAll(t, setup.stdout.String(), "Interrupted")
}

func TestRunCmd_Run_SourceDirOverride(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	ts.env.SetSourceDir(t.TempDir())

	dir := t.TempDir()
	taskFile := testutil.WriteTaskFile(t, filepath.Join(dir, "task.yaml"), &config.Task{
		Name:   "no-source",
		Source: config.Source{PromptFile: filepath.Join(dir, "prompt.md")},
	})
	testutil.WriteFile(t, filepath.Join(dir, "prompt.md"), "do the thing")

	cmd := &cmdRun{taskPath: taskFile, detach: true}

	assert.ErrVal(t, cmd.Run(ts.env)).Nil().Fatal()
	assert.ContainsAll(t, ts.stdout.String(), "started")
}

func TestRunCmd_Run_PromptFileOutsideTaskDir(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)

	dir := t.TempDir()
	srcDir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "outside.md"), "external prompt body")
	taskFile := testutil.WriteTaskFile(t, filepath.Join(dir, "tasks", "foo.yaml"), &config.Task{
		Name: "outside-prompt",
		Source: config.Source{
			LocalPath:  srcDir,
			PromptFile: "../outside.md",
		},
	})

	cmd := &cmdRun{taskPath: taskFile, detach: true}
	assert.ErrVal(t, cmd.Run(ts.env)).Nil().Fatal()

	ws := ts.state.Data[cmd.createdID]
	// TODO: use assert.Val(t, ws).NotZero().Fatal() once pkg/assert has it; a nil ws panics on the next line.
	assert.NotNil(t, ws)
	assert.PathExists(t, filepath.Join(ws.BasePath, "workspace", "outside.md"))

	cfg := testutil.ReadFile(t, filepath.Join(ws.BasePath, "config.yaml"))
	assert.Contains(t, cfg, "prompt_file: outside.md")
}

func TestRunCmd_Run_ModelOverride(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	ts.env.SetModel("custom-model")

	cmd := &cmdRun{taskPath: writeTaskFile(t, t.TempDir(), "model-test"), detach: true}

	assert.ErrVal(t, cmd.Run(ts.env)).Nil().Fatal()
	assert.ContainsAll(t, ts.stdout.String(), "started")
}

func TestRunCmd_Run_SourceDirNotUsedWhenYAMLHasSource(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	ts.env.SetSourceDir("/should/not/be/used")

	cmd := &cmdRun{taskPath: writeTaskFile(t, t.TempDir(), "has-source"), detach: true}

	assert.ErrVal(t, cmd.Run(ts.env)).Nil().Fatal()
	assert.ContainsAll(t, ts.stdout.String(), "started")
}

func TestRunCmd_Run_MissingFile(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	cmd := &cmdRun{taskPath: "/no/such/file.yaml", detach: true}

	err := cmd.Run(ts.env)

	assert.ErrVal(t, err).Is(os.ErrNotExist).Fatal()
}

func TestRunCmd_Run_ContinuePolicy(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)

	dir := t.TempDir()
	srcDir := t.TempDir()
	ws := testfake.CreateWorkspace(t, dir, "ws-cont", "continue-test", "completed")
	setup.state.Data["ws-cont"] = ws

	taskFile := filepath.Join(t.TempDir(), "task.yaml")
	testutil.WriteFile(t, taskFile, fmt.Sprintf(
		"name: continue-test\ncontinue_policy: resume\nsource:\n  local_path: %s\n", srcDir,
	))

	cmd := &cmdRun{taskPath: taskFile, detach: true}
	assert.ErrVal(t, cmd.Run(setup.env)).Nil().Fatal()
	assert.Contains(t, setup.stdout.String(), "Workspace resumed: continue-test")
}

func TestCmdRun_Run_SkipStartIfDone(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)

	taskFile := createDoneWorkspaceTask(t, setup)
	cmd := &cmdRun{taskPath: taskFile, detach: true}
	err := cmd.Run(setup.env)

	assert.ErrVal(t, err).Is(workspace.ErrWorkspaceDone).Fatal()
	out := setup.stdout.String()
	assert.Contains(t, out, "Workspace already done")
	assert.Contains(t, out, ".dwoe-done")
}

func createTestTask(t *testing.T, logs string, exitCode int) (*cmdTestSetup, string) {
	t.Helper()
	setup := newCmdTestSetup(t)
	setup.docker.ContainerLogsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		return logReader(logs), nil
	}
	setup.docker.WaitContainerFn = func(_ context.Context, _ string) (int, error) {
		return exitCode, nil
	}
	dir := t.TempDir()
	taskFile := writeTaskFile(t, dir, "test-task")
	return setup, taskFile
}
