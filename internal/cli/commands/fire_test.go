// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestFireCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		args      []string
		wantRepo  string
		wantWork  string
		wantDo    string
		wantModel string
		wantBatch string
	}{
		{"long_flags", []string{"--repo", "/tmp/repo", "--work", "task.md"}, "/tmp/repo", "task.md", "", "", ""},
		{"short_flags", []string{"-r", "/tmp/repo", "-w", "task.md"}, "/tmp/repo", "task.md", "", "", ""},
		{"with_model", []string{"-r", "/tmp/repo", "-w", "task.md", "-m", "claude-sonnet-4-6"}, "/tmp/repo", "task.md", "", "claude-sonnet-4-6", ""},
		{"with_batch", []string{"-r", "/tmp/repo", "-w", "task.md", "-b", "my-batch"}, "/tmp/repo", "task.md", "", "", "my-batch"},
		{"with_do", []string{"-r", "/tmp/repo", "--do", "fix it"}, "/tmp/repo", "", "fix it", "", ""},
		{"do_only", []string{"--repo", "/tmp/repo", "--do", "merge patches"}, "/tmp/repo", "", "merge patches", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := new(cmdFire)
			assert.NotErr(t, cmd.Parse(tc.args))
			assert.Equal(t, cmd.repo, tc.wantRepo)
			assert.Equal(t, cmd.work, tc.wantWork)
			assert.Equal(t, cmd.do, tc.wantDo)
			assert.Equal(t, cmd.model, tc.wantModel)
			assert.Equal(t, cmd.batchID, tc.wantBatch)
		})
	}
}

func TestFireCmd_Parse_DoAndWorkConflict(t *testing.T) {
	t.Parallel()
	cmd := new(cmdFire)
	assert.Err(t, cmd.Parse([]string{"-r", "/tmp/repo", "--do", "fix", "-w", "task.md"}))
}

func TestFireCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdFire))
}

func TestFireCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("detach", func(t *testing.T) {
		t.Parallel()
		setup := newCmdTestSetup(t)
		srcDir := t.TempDir()
		workFile := filepath.Join(t.TempDir(), "task.md")
		testutil.WriteFile(t, workFile, "do the thing")

		cmd := &cmdFire{repo: srcDir, work: workFile}
		assert.NotErr(t, cmd.Run(setup.env))
		assert.ContainsAll(t, setup.stdout.String(), "Started workspace:", "View logs:")
	})

	t.Run("with_do", func(t *testing.T) {
		t.Parallel()
		setup := newCmdTestSetup(t)
		srcDir := t.TempDir()

		cmd := &cmdFire{repo: srcDir, do: "just do it"}
		assert.NotErr(t, cmd.Run(setup.env))
		assert.ContainsAll(t, setup.stdout.String(), "Started workspace:", "View logs:")
	})

	t.Run("with_batch", func(t *testing.T) {
		t.Parallel()
		setup := newCmdTestSetup(t)
		srcDir := t.TempDir()
		workFile := filepath.Join(t.TempDir(), "task.md")
		testutil.WriteFile(t, workFile, "do the thing")

		cmd := &cmdFire{repo: srcDir, work: workFile, batchID: "test-batch"}
		assert.NotErr(t, cmd.Run(setup.env))
		assert.ContainsAll(t, setup.stdout.String(), "Started workspace:", "Batch: test-batch")
	})
}

func TestFireCmd_ResolveWork_SingleFile(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "task.md"), "do the thing")

	// changing directory to test that resolveWork returns an absolute path
	// when a relative task path is provided which avoid the issue of not being
	// able to copy the prompt into workspace where containerized agent can
	// access it
	t.Chdir(dir)

	got, err := resolveWork("./task.md")

	assert.NotErr(t, err)
	assert.Condition(t, filepath.IsAbs(got))
	data, _ := os.ReadFile(got)
	assert.Equal(t, string(data), "do the thing")
}

func TestFireCmd_ResolveWork_Directory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "a.md"), "first")
	testutil.WriteFile(t, filepath.Join(dir, "b.md"), "second")

	got, err := resolveWork(dir)

	assert.NotErr(t, err)
	data, _ := os.ReadFile(got)
	content := string(data)
	assert.ContainsAll(t, content, "a.md", "first", "b.md", "second")
}

func TestFireCmd_IsRepoURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"/tmp/repo", false},
		{"./relative/path", false},
		{"../parent", false},
		{"https://github.com/user/repo.git", true},
		{"http://github.com/user/repo", true},
		{"git@github.com:user/repo.git", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, isRepoURL(tc.input), tc.want)
		})
	}
}
