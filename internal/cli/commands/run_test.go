// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

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
			if err := cmd.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			if cmd.taskPath != tc.wantPath {
				t.Errorf("taskPath = %q, want %q", cmd.taskPath, tc.wantPath)
			}
			if cmd.name != tc.wantName {
				t.Errorf("name = %q, want %q", cmd.name, tc.wantName)
			}
			if cmd.detach != tc.wantDetach {
				t.Errorf("detach = %v, want %v", cmd.detach, tc.wantDetach)
			}
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

		err := cmd.Run(ts.env)

		if err != nil {
			t.Fatal(err)
		}
		testutil.ContainsAll(t, ts.stdout.String(), "Started workspace:", "Status: running")
	})

	t.Run("attached_success", func(t *testing.T) {
		t.Parallel()
		ts, taskFile := createTestTask(t, "line1\nline2\n<promise>DONE</promise>\n", 0)
		cmd := &cmdRun{taskPath: taskFile}

		err := cmd.Run(ts.env)

		if err != nil {
			t.Fatal(err)
		}
		testutil.ContainsAll(t, ts.stdout.String(), "line1", "line2", "completed")
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
		if exitErr.code != 1 {
			t.Errorf("exitErr.code = %d, want 1", exitErr.code)
		}
		testutil.ContainsAll(t, ts.stdout.String(), "failed")
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

	testutil.WantErr(t, err, errRunInterrupted)
	testutil.ContainsAll(t, setup.stdout.String(), "Interrupted")
}

func TestRunCmd_Run_MissingFile(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	cmd := &cmdRun{taskPath: "/no/such/file.yaml", detach: true}

	err := cmd.Run(ts.env)

	testutil.WantErr(t, err, os.ErrNotExist)
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
