// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestMain(m *testing.M) {
	_, err := exec.Command("git", "version").CombinedOutput()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "git unavailable: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestGit_CloneRepo(t *testing.T) {
	t.Parallel()
	repoDir := t.TempDir()
	err := CloneRepo("https://github.com/i-zaitsev/url.git", repoDir, "main")
	assert.NotErr(t, err)
}

func TestGit_CopyLocalDir(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	err := CopyLocalDir(srcDir, dstDir)
	assert.NotErr(t, err)
}

func TestGit_CloneAndCopy(t *testing.T) {
	t.Parallel()
	cloneDir := t.TempDir()
	copyDir := t.TempDir()

	assert.NotErr(t, CloneRepo("https://github.com/i-zaitsev/url.git", cloneDir, "main"))
	assert.NotErr(t, CopyLocalDir(cloneDir, copyDir))

	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); err != nil {
		t.Fatal("clone missing .git")
	}
	if _, err := os.Stat(filepath.Join(copyDir, ".git")); err == nil {
		t.Fatal("copy should not have .git")
	}

	for _, f := range []string{"go.mod", "url.go"} {
		src, err := os.ReadFile(filepath.Join(cloneDir, f))
		if err != nil {
			t.Fatalf("read clone/%s: %v", f, err)
		}
		dst, err := os.ReadFile(filepath.Join(copyDir, f))
		if err != nil {
			t.Fatalf("read copy/%s: %v", f, err)
		}
		if string(src) != string(dst) {
			t.Fatalf("%s content mismatch", f)
		}
	}
}

func TestGit_CloneCopyErrors(t *testing.T) {
	t.Parallel()
	tests := map[string]func(dir string) error{
		"clone/invalid_url": func(dir string) error {
			return CloneRepo("not-a-url", dir, "main")
		},
		"clone/bad_repo": func(dir string) error {
			return CloneRepo("https://github.com/i-zaitsev/nonexistent-xxx.git", dir, "main")
		},
		"copy/bad_src": func(dir string) error {
			return CopyLocalDir("/no/such/dir", dir)
		},
	}
	for name, testFn := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			assert.Err(t, testFn(tmpDir))
		})
	}
}

func TestGit_InitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := map[string]string{
		"user.name":  "Test User",
		"user.email": "test@example.com",
	}
	assert.NotErr(t, InitRepo(dir, cfg["user.name"], cfg["user.email"]))
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatal("missing .git")
	}
	for k, want := range cfg {
		got, _ := exec.Command("git", "-C", dir, "config", k).CombinedOutput()
		if strings.TrimSpace(string(got)) != want {
			t.Fatalf("config %s: got %q, want %q", k, got, want)
		}
	}
}

func TestCollect(t *testing.T) {
	t.Parallel()

	wsDir := initTestRepo(t, "Agent", "agent@test.dev")
	addCommit(t, wsDir, "feature.go", "package main", "Add feature")
	addCommit(t, wsDir, "feature_test.go", "package main", "Add tests")

	targetDir := initTestRepo(t, "Owner", "owner@test.dev")

	n, err := Collect(wsDir, targetDir, "agent/feature")
	assert.NotErr(t, err)
	assert.Equal(t, n, 2)

	// Verify branch exists with the commits.
	out, _ := exec.Command("git", "-C", targetDir, "log", "--oneline", "agent/feature").CombinedOutput()
	assert.Contains(t, string(out), "Add feature")
	assert.Contains(t, string(out), "Add tests")

	// Verify we're back on main.
	branch, _ := exec.Command("git", "-C", targetDir, "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
	assert.Equal(t, strings.TrimSpace(string(branch)), "main")
}

func TestCollect_NoAgentCommits(t *testing.T) {
	t.Parallel()

	wsDir := initTestRepo(t, "Agent", "agent@test.dev")
	targetDir := initTestRepo(t, "Owner", "owner@test.dev")

	n, err := Collect(wsDir, targetDir, "agent/empty")
	assert.NotErr(t, err)
	assert.Zero(t, n)
}

func TestCollect_BranchExists(t *testing.T) {
	t.Parallel()

	wsDir := initTestRepo(t, "Agent", "agent@test.dev")
	addCommit(t, wsDir, "f.go", "package f", "Work")

	targetDir := initTestRepo(t, "Owner", "owner@test.dev")
	mustGit(t, "-C", targetDir, "checkout", "-b", "taken")
	mustGit(t, "-C", targetDir, "checkout", "main")

	_, err := Collect(wsDir, targetDir, "taken")
	assert.Err(t, err)
}

func TestEnsureRepoReady_AlreadyInit(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t, "Test", "test@test.dev")

	assert.NotErr(t, EnsureRepoReady(dir, "Test", "test@test.dev"))

	out, _ := exec.Command("git", "-C", dir, "log", "--oneline").CombinedOutput()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	assert.Equal(t, len(lines), 1)
}

func TestEnsureRepoReady_NewRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "main.go"), "package main")

	assert.NotErr(t, EnsureRepoReady(dir, "Test", "test@test.dev"))

	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatal("expected .git to exist")
	}

	out, _ := exec.Command("git", "-C", dir, "log", "--oneline").CombinedOutput()
	assert.Contains(t, string(out), "initial")
}

func TestMergeBranches(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t, "Test", "test@test.dev")

	mustGit(t, "-C", dir, "checkout", "-b", "branch-a")
	addCommit(t, dir, "a.txt", "a", "add a")
	mustGit(t, "-C", dir, "checkout", "main")

	mustGit(t, "-C", dir, "checkout", "-b", "branch-b")
	addCommit(t, dir, "b.txt", "b", "add b")
	mustGit(t, "-C", dir, "checkout", "main")

	assert.NotErr(t, MergeBranches(dir, []string{"branch-a", "branch-b"}, "merge features"))

	for _, f := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected %s after merge", f)
		}
	}
}

func TestMergeBranches_Empty(t *testing.T) {
	t.Parallel()
	assert.NotErr(t, MergeBranches("/nonexistent", nil, "msg"))
}

func TestExportPatches(t *testing.T) {
	t.Parallel()

	wsDir := initTestRepo(t, "Agent", "agent@test.dev")
	addCommit(t, wsDir, "feature.go", "package main", "Add feature")
	addCommit(t, wsDir, "feature_test.go", "package main", "Add tests")

	outDir := filepath.Join(t.TempDir(), "patches")
	n, err := ExportPatches(wsDir, outDir)
	assert.NotErr(t, err)
	assert.Equal(t, n, 2)

	patches, _ := filepath.Glob(filepath.Join(outDir, "*.patch"))
	assert.Equal(t, len(patches), 2)

	for _, p := range patches {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read patch %s: %v", p, err)
		}
		if len(data) == 0 {
			t.Fatalf("patch %s is empty", p)
		}
	}
}

func TestExportPatches_NoAgentCommits(t *testing.T) {
	t.Parallel()

	wsDir := initTestRepo(t, "Agent", "agent@test.dev")

	outDir := filepath.Join(t.TempDir(), "patches")
	n, err := ExportPatches(wsDir, outDir)
	assert.NotErr(t, err)
	assert.Zero(t, n)
}

func TestExportPatches_PatchesApplyCleanly(t *testing.T) {
	t.Parallel()

	wsDir := initTestRepo(t, "Agent", "agent@test.dev")
	addCommit(t, wsDir, "new.go", "package new", "Add new file")

	outDir := filepath.Join(t.TempDir(), "patches")
	n, err := ExportPatches(wsDir, outDir)
	assert.NotErr(t, err)
	assert.Equal(t, n, 1)

	targetDir := initTestRepo(t, "Owner", "owner@test.dev")

	patches, _ := filepath.Glob(filepath.Join(outDir, "*.patch"))
	args := append([]string{"-C", targetDir, "am"}, patches...)
	mustGit(t, args...)

	if _, err := os.Stat(filepath.Join(targetDir, "new.go")); err != nil {
		t.Fatal("expected new.go after applying patches")
	}

	out, _ := exec.Command("git", "-C", targetDir, "log", "--oneline").CombinedOutput()
	assert.Contains(t, string(out), "Add new file")
}

func TestValidateGitURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://github.com/user/repo.git", false},
		{"http://github.com/user/repo.git", false},
		{"https://github.com/user/repo", false},
		{"git@github.com:user/repo.git", false},
		{"git@gitlab.com:group/sub/repo.git", false},
		{"user@host:path", false},
		{"", true},
		{"not-a-url", true},
		{"/some/local/path", true},
		{"ftp://example.com/repo.git", true},
		{"just-words", true},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			err := validateGitURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGitURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestWalkDir(t *testing.T) {
	t.Parallel()

	t.Run("empty_dir", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		files, err := walkDir(dir, ".git")
		assert.NotErr(t, err)
		assert.Zero(t, len(files))
	})

	t.Run("nested_files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.WriteFile(t, filepath.Join(dir, "a.txt"), "a")
		testutil.WriteFile(t, filepath.Join(dir, "sub", "b.txt"), "b")

		files, err := walkDir(dir)
		assert.NotErr(t, err)
		assert.Equal(t, len(files), 2)
		want := map[string]bool{"a.txt": true, filepath.Join("sub", "b.txt"): true}
		for _, f := range files {
			if !want[f] {
				t.Errorf("unexpected file: %s", f)
			}
		}
	})

	t.Run("excludes_git", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.WriteFile(t, filepath.Join(dir, "file.txt"), "ok")
		testutil.WriteFile(t, filepath.Join(dir, ".git", "config"), "hidden")
		testutil.WriteFile(t, filepath.Join(dir, ".git", "objects", "x"), "obj")

		files, err := walkDir(dir, ".git")
		assert.NotErr(t, err)
		assert.Equal(t, len(files), 1)
		assert.Equal(t, files[0], "file.txt")
	})

	t.Run("nonexistent_dir", func(t *testing.T) {
		t.Parallel()
		_, err := walkDir("/no/such/dir")
		assert.Err(t, err)
	})

	t.Run("file_not_dir", func(t *testing.T) {
		t.Parallel()
		f := filepath.Join(t.TempDir(), "file.txt")
		testutil.WriteFile(t, f, "x")
		_, err := walkDir(f)
		assert.Err(t, err)
	})
}

func TestGit_InitRepoErrors(t *testing.T) {
	t.Parallel()
	emptyCases := []struct {
		testName  string
		userName  string
		userEmail string
	}{
		{
			testName:  "empty_user_name",
			userName:  "",
			userEmail: "test@example.com",
		},
		{
			testName:  "empty_user_email",
			userName:  "Test User",
			userEmail: "",
		},
		{
			testName:  "empty_user_name_and_email",
			userName:  "",
			userEmail: "",
		},
	}
	for _, tc := range emptyCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			assert.Err(t, InitRepo(t.TempDir(), tc.userName, tc.userEmail))
		})
	}
}

func mustGit(t *testing.T, args ...string) {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func initTestRepo(t *testing.T, name, email string) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, "init", dir, "--initial-branch", "main")
	mustGit(t, "-C", dir, "config", "user.name", name)
	mustGit(t, "-C", dir, "config", "user.email", email)
	testutil.WriteFile(t, filepath.Join(dir, "file.txt"), "original")
	mustGit(t, "-C", dir, "add", ".")
	mustGit(t, "-C", dir, "commit", "-m", "Initial commit")
	return dir
}

func addCommit(t *testing.T, dir, file, content, msg string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(dir, file), content)
	mustGit(t, "-C", dir, "add", ".")
	mustGit(t, "-C", dir, "commit", "-m", msg)
}
