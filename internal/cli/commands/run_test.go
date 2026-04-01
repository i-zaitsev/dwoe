// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/testutil"
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
			assert.NotErr(t, cmd.Parse(tc.args))
			assert.Equal(t, cmd.taskPath, tc.wantPath)
			assert.Equal(t, cmd.name, tc.wantName)
			assert.Equal(t, cmd.detach, tc.wantDetach)
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
		assert.NotErr(t, cmd.Run(ts.env))
		assert.ContainsAll(t, ts.stdout.String(), "Started workspace:", "Status: running")
	})

	t.Run("attached_success", func(t *testing.T) {
		t.Parallel()
		ts, taskFile := createTestTask(t, "line1\nline2\n<promise>DONE</promise>\n", 0)
		cmd := &cmdRun{taskPath: taskFile}
		assert.NotErr(t, cmd.Run(ts.env))
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
		assert.Equal(t, exitErr.code, 1)
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

	assert.ErrIs(t, err, errRunInterrupted)
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

	assert.NotErr(t, cmd.Run(ts.env))
	assert.ContainsAll(t, ts.stdout.String(), "Started workspace:")
}

func TestRunCmd_Run_ModelOverride(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	ts.env.SetModel("custom-model")

	cmd := &cmdRun{taskPath: writeTaskFile(t, t.TempDir(), "model-test"), detach: true}

	assert.NotErr(t, cmd.Run(ts.env))
	assert.ContainsAll(t, ts.stdout.String(), "Started workspace:")
}

func TestRunCmd_Run_SourceDirNotUsedWhenYAMLHasSource(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	ts.env.SetSourceDir("/should/not/be/used")

	cmd := &cmdRun{taskPath: writeTaskFile(t, t.TempDir(), "has-source"), detach: true}

	assert.NotErr(t, cmd.Run(ts.env))
	assert.ContainsAll(t, ts.stdout.String(), "Started workspace:")
}

func TestRunCmd_Run_MissingFile(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	cmd := &cmdRun{taskPath: "/no/such/file.yaml", detach: true}

	err := cmd.Run(ts.env)

	assert.ErrIs(t, err, os.ErrNotExist)
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
