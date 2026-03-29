// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestBatchCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		wantDir string
	}{
		{"dir_only", []string{"./examples"}, "./examples"},
		{"absolute", []string{"/tmp/tasks"}, "/tmp/tasks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := new(cmdBatch)
			if err := cmd.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			if cmd.dir != tt.wantDir {
				t.Errorf("dir = %q, want %q", cmd.dir, tt.wantDir)
			}
		})
	}
}

func TestBatchCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdBatch))
}

func TestBatchCmd_Run(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	srcDir := t.TempDir()
	batchDir := t.TempDir()
	writeBatchTaskFile(t, batchDir, "alpha", srcDir)
	writeBatchTaskFile(t, batchDir, "beta", srcDir)

	cmd := &cmdBatch{
		dir: batchDir,
		loadConfig: func(taskPath, _ string) (*config.Task, error) {
			return &config.Task{
				Name:   filepath.Base(taskPath),
				Source: config.Source{LocalPath: srcDir},
			}, nil
		},
		ensureRepo: func(_, _, _ string) error { return nil },
	}

	err := cmd.Run(setup.env)
	if err != nil {
		t.Fatal(err)
	}

	out := setup.stdout.String()
	testutil.ContainsAll(t, out,
		"discovered 2 task(s)",
		"Batch ID:",
		"Summary: 2 total, 2 completed, 0 failed",
	)
}

func TestBatchCmd_Run_PartialFailure(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	srcDir := t.TempDir()
	batchDir := t.TempDir()
	writeBatchTaskFile(t, batchDir, "alpha", srcDir)
	writeBatchTaskFile(t, batchDir, "beta", srcDir)

	var waitCalls int
	setup.docker.WaitContainerFn = func(_ context.Context, _ string) (int, error) {
		waitCalls++
		if waitCalls == 1 {
			return 1, nil
		}
		return 0, nil
	}

	cmd := &cmdBatch{
		dir: batchDir,
		loadConfig: func(taskPath, _ string) (*config.Task, error) {
			return &config.Task{
				Name:   filepath.Base(taskPath),
				Source: config.Source{LocalPath: srcDir},
			}, nil
		},
		ensureRepo: func(_, _, _ string) error { return nil },
	}

	err := cmd.Run(setup.env)
	if err == nil {
		t.Fatal("expected error for partial failure")
	}

	out := setup.stdout.String()
	testutil.ContainsAll(t, out,
		"Summary: 2 total,",
		"1 failed",
	)
}

func TestDiscoverTasks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{"task-a.yaml", "task-b.yaml", "task-c.yaml"} {
		testutil.WriteFile(t, filepath.Join(dir, name), "name: test")
	}

	// Non-matching file should be ignored.
	testutil.WriteFile(t, filepath.Join(dir, "config.yaml"), "x: y")

	got, err := discoverTasks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, name := range []string{"task-a.yaml", "task-b.yaml", "task-c.yaml"} {
		if filepath.Base(got[i]) != name {
			t.Errorf("got[%d] = %q, want %q", i, filepath.Base(got[i]), name)
		}
	}
}

func TestDiscoverTasks_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := discoverTasks(dir)
	if err == nil {
		t.Fatal("expected error for empty dir")
	}
}
