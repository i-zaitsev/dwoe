// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"os"
	"path/filepath"
	"testing"

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
			if err := cmd.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			if cmd.repo != tc.wantRepo {
				t.Errorf("repo = %q, want %q", cmd.repo, tc.wantRepo)
			}
			if cmd.work != tc.wantWork {
				t.Errorf("work = %q, want %q", cmd.work, tc.wantWork)
			}
			if cmd.do != tc.wantDo {
				t.Errorf("do = %q, want %q", cmd.do, tc.wantDo)
			}
			if cmd.model != tc.wantModel {
				t.Errorf("model = %q, want %q", cmd.model, tc.wantModel)
			}
			if cmd.batchID != tc.wantBatch {
				t.Errorf("batchID = %q, want %q", cmd.batchID, tc.wantBatch)
			}
		})
	}
}

func TestFireCmd_Parse_DoAndWorkConflict(t *testing.T) {
	t.Parallel()
	cmd := new(cmdFire)
	err := cmd.Parse([]string{"-r", "/tmp/repo", "--do", "fix", "-w", "task.md"})
	if err == nil {
		t.Fatal("expected error when both --do and --work are set")
	}
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
		err := cmd.Run(setup.env)

		if err != nil {
			t.Fatal(err)
		}
		testutil.ContainsAll(t, setup.stdout.String(), "Started workspace:", "View logs:")
	})

	t.Run("with_do", func(t *testing.T) {
		t.Parallel()
		setup := newCmdTestSetup(t)
		srcDir := t.TempDir()

		cmd := &cmdFire{repo: srcDir, do: "just do it"}
		err := cmd.Run(setup.env)

		if err != nil {
			t.Fatal(err)
		}
		testutil.ContainsAll(t, setup.stdout.String(), "Started workspace:", "View logs:")
	})

	t.Run("with_batch", func(t *testing.T) {
		t.Parallel()
		setup := newCmdTestSetup(t)
		srcDir := t.TempDir()
		workFile := filepath.Join(t.TempDir(), "task.md")
		testutil.WriteFile(t, workFile, "do the thing")

		cmd := &cmdFire{repo: srcDir, work: workFile, batchID: "test-batch"}
		err := cmd.Run(setup.env)

		if err != nil {
			t.Fatal(err)
		}
		testutil.ContainsAll(t, setup.stdout.String(), "Started workspace:", "Batch: test-batch")
	})
}

func TestFireCmd_ResolveWork(t *testing.T) {
	t.Parallel()

	t.Run("single_file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		f := filepath.Join(dir, "task.md")
		testutil.WriteFile(t, f, "do the thing")

		got, err := resolveWork(f)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(got)
		if string(data) != "do the thing" {
			t.Errorf("content = %q, want %q", string(data), "do the thing")
		}
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.WriteFile(t, filepath.Join(dir, "a.md"), "first")
		testutil.WriteFile(t, filepath.Join(dir, "b.md"), "second")

		got, err := resolveWork(dir)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(got)
		content := string(data)
		testutil.ContainsAll(t, content, "a.md", "first", "b.md", "second")
	})
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
			if got := isRepoURL(tc.input); got != tc.want {
				t.Errorf("isRepoURL(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
