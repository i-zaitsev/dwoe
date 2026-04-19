// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package workspace

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneRepo shallow-clones a git repository into a localPath at the given branch.
func CloneRepo(repoURL, localPath, branch string) error {
	slog.Info("git: clone", "url", repoURL, "path", localPath, "branch", branch)
	if err := validateGitURL(repoURL); err != nil {
		return fmt.Errorf("clone: %w", err)
	}
	return git("clone", "--depth", "1", "--branch", branch, repoURL, localPath)
}

// CopyLocalDir copies all tracked files from src to dst, excluding .git.
func CopyLocalDir(src, dst string) error {
	slog.Info("git: copy", "src", src, "dst", dst)
	files, err := walkDir(src, ".git")
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	var errs []error
	for _, rel := range files {
		errs = append(errs, func() error {
			srcFile := filepath.Join(src, rel)

			dstFile := filepath.Join(dst, rel)
			if errDir := os.MkdirAll(filepath.Dir(dstFile), 0o755); errDir != nil {
				return fmt.Errorf("copy: mkdir: %w", errDir)
			}
			info, errStat := os.Stat(srcFile)
			if errStat != nil {
				return fmt.Errorf("copy: stat: %w", errStat)
			}
			fIn, errIn := os.Open(srcFile)
			if errIn != nil {
				return fmt.Errorf("copy: open: %w", errIn)
			}
			defer func() {
				_ = fIn.Close()
			}()
			fOut, errOut := os.OpenFile(dstFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
			if errOut != nil {
				return fmt.Errorf("copy: open: %w", errOut)
			}
			if _, errCopy := io.Copy(fOut, fIn); errCopy != nil {
				defer func() {
					_ = fOut.Close()
				}()
				return fmt.Errorf("copy: %s -> %s: %w", srcFile, dstFile, errCopy)
			}
			return fOut.Close()
		}())
	}
	return errors.Join(errs...)
}

// InitRepo initializes a new git repository with the given identity.
func InitRepo(repoPath, userName, userEmail string) error {
	slog.Info("git: init", "path", repoPath)
	if userName == "" || userEmail == "" {
		return fmt.Errorf("init: username and email cannot be empty")
	}
	commands := [][]string{
		{"init", repoPath, "--initial-branch", "main", "--quiet"},
		{"-C", repoPath, "config", "user.name", userName},
		{"-C", repoPath, "config", "user.email", userEmail},
	}
	for _, cmd := range commands {
		if err := git(cmd...); err != nil {
			return fmt.Errorf("init: %w", err)
		}
	}
	return nil
}

// EnsureRepoReady initializes and commits all files if no .git directory exists.
func EnsureRepoReady(repoPath, userName, userEmail string) error {
	slog.Info("git: ensure-repo-ready", "path", repoPath)
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil
	}
	if err := InitRepo(repoPath, userName, userEmail); err != nil {
		return err
	}
	if err := git("-C", repoPath, "add", "."); err != nil {
		return fmt.Errorf("ensure repo: add: %w", err)
	}
	return git("-C", repoPath, "commit", "-m", "initial", "--allow-empty")
}

// MergeBranches merges the given branches into the current branch of repoPath.
func MergeBranches(repoPath string, branches []string, message string) error {
	if len(branches) == 0 {
		return nil
	}
	slog.Info("git: merge", "repo", repoPath, "branches", branches)
	args := append([]string{"-C", repoPath, "merge"}, branches...)
	args = append(args, "-m", message)
	return git(args...)
}

// Collect applies workspace commits as patches onto a new branch in targetRepo.
func Collect(workspaceDir, targetRepo, branch string) (int, error) {
	slog.Info("git: collect", "workspace", workspaceDir, "repo", targetRepo, "branch", branch)

	tmpDir, err := os.MkdirTemp("", "dwoe-patches-*")
	if err != nil {
		return 0, fmt.Errorf("collect: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	patches, err := formatPatches(workspaceDir, tmpDir)
	if err != nil {
		return 0, fmt.Errorf("collect: %w", err)
	}
	if len(patches) == 0 {
		return 0, nil
	}

	if err := git("-C", targetRepo, "checkout", "-b", branch); err != nil {
		return 0, fmt.Errorf("collect: %w", err)
	}

	args := append([]string{"-C", targetRepo, "am"}, patches...)
	if err := git(args...); err != nil {
		_ = git("-C", targetRepo, "am", "--abort")
		_ = git("-C", targetRepo, "checkout", "-")
		_ = git("-C", targetRepo, "branch", "-D", branch)
		return 0, fmt.Errorf("collect: %w", err)
	}

	if err := git("-C", targetRepo, "checkout", "-"); err != nil {
		return 0, fmt.Errorf("collect: %w", err)
	}

	return len(patches), nil
}

// ExportPatches writes git format-patch files from repoDir into outDir.
func ExportPatches(repoDir, outDir string) (int, error) {
	slog.Info("git: export-patches", "repo", repoDir, "outDir", outDir)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, fmt.Errorf("export-patches: %w", err)
	}

	patches, err := formatPatches(repoDir, outDir)
	if err != nil {
		return 0, fmt.Errorf("export-patches: %w", err)
	}
	return len(patches), nil
}

func formatPatches(repoDir, outDir string) ([]string, error) {
	base, err := gitOutput("-C", repoDir, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return nil, err
	}
	if err := git("-C", repoDir, "format-patch", base, "-o", outDir); err != nil {
		return nil, err
	}
	return filepath.Glob(filepath.Join(outDir, "*.patch"))
}

func git(args ...string) error {
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v: %w\n%s", args, err, string(out))
	}
	return nil
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %v: %w\n%s", args, err, string(out))
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")[0], nil
}

func validateGitURL(rawURL string) error {
	if strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "http://") {
		return nil
	}
	if i := strings.Index(rawURL, "@"); i >= 0 && strings.Contains(rawURL[i:], ":") {
		return nil
	}
	return fmt.Errorf("invalid git URL: %s", rawURL)
}

func walkDir(root string, excludes ...string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}
	var files []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		name := filepath.Base(rel)
		if d.IsDir() {
			for _, ex := range excludes {
				if name == ex {
					return filepath.SkipDir
				}
			}
			return nil
		}
		for _, ex := range excludes {
			if name == ex {
				return nil
			}
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}

func gitFullOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %v: %w\n%s", args, err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}
