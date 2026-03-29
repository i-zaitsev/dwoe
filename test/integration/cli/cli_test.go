// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build integration

// Package cli_test contains integration tests that exercise the dwoe CLI
// binary against real Docker containers. Each test builds and invokes the
// compiled binary, verifying end-to-end behavior of workspace commands.
package cli_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/docker"
	"github.com/i-zaitsev/dwoe/internal/log"
	"github.com/i-zaitsev/dwoe/internal/testutil"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

const (
	basicImage   = "dwoe-cli-test-basic:latest"
	gitWorkImage = "dwoe-cli-test-gitwork:latest"
)

var binaryPath string

// Builds test Docker images and the CLI binary into a temp directory.
// An immediately invoked closure ensures defers clean up after m.Run returns but before os.Exit.
// The tests expect docker service is running.
func TestMain(m *testing.M) {
	var code int
	func() {
		opts := log.DefaultOpts()
		opts.Level = 8
		log.Setup(opts)

		ctx := context.Background()

		client, err := docker.NewClient()
		fatal(err)
		defer client.Close()

		fatal(client.Ping(ctx))
		fatal(client.BuildImage(ctx, "../testdata/Dockerfile.basic", basicImage, os.Stderr))
		fatal(client.BuildImage(ctx, "../testdata/Dockerfile.gitwork", gitWorkImage, os.Stderr))

		tmpDir, err := os.MkdirTemp("", "dwoe-cli-test-bin-*")
		fatal(err)
		defer func() {
			if errRemove := os.RemoveAll(tmpDir); errRemove != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to delete the temp dir: %s", tmpDir)
			}
		}()

		binaryPath = filepath.Join(tmpDir, "dwoe")
		build := exec.Command("go", "build", "-o", binaryPath, "../../../cmd/dwoe")
		build.Stderr = os.Stderr
		fatal(build.Run())

		code = m.Run()
	}()
	os.Exit(code)
}

// Runs a task in attached mode and verifies that the workspace starts,
// produces expected output, completes, and exits with code 0.
func TestCLI_RunAttached(t *testing.T) {
	dataDir := t.TempDir()
	t.Cleanup(func() { destroyAll(t, dataDir) })

	taskFile := writeTaskYAML(t, t.TempDir(), "run-attached", basicImage, t.TempDir())
	out := runCLI(t, cliArgs(dataDir, "run", taskFile)...)

	testutil.ContainsAll(t, out, "Started workspace:", "test ok", "completed", "exit code 0")
}

// Exercises the full workspace lifecycle through individual CLI commands:
// create (pending) → start (running) → status → stop → destroy → list (empty).
// Verifies the basic lifecycle of a workspace.
func TestCLI_CreateStartStopDestroy(t *testing.T) {
	dataDir := t.TempDir()
	t.Cleanup(func() { destroyAll(t, dataDir) })

	taskFile := writeTaskYAML(t, t.TempDir(), "lifecycle", basicImage, t.TempDir())

	out := runCLI(t, cliArgs(dataDir, "create", taskFile)...)
	testutil.ContainsAll(t, out, "Created workspace:", "Status: pending")
	id := parseID(t, out)

	out = runCLI(t, cliArgs(dataDir, "start", id)...)
	testutil.ContainsAll(t, out, "Started workspace:", "Status: running")

	out = runCLI(t, cliArgs(dataDir, "status", id)...)
	testutil.ContainsAll(t, out, "Status:")

	out = runCLI(t, cliArgs(dataDir, "stop", "-f", id)...)
	testutil.ContainsAll(t, out, "Workspace stopped.")

	out = runCLI(t, cliArgs(dataDir, "destroy", id)...)
	testutil.ContainsAll(t, out, "Workspace destroyed.")

	out = runCLI(t, cliArgs(dataDir, "list")...)
	testutil.ContainsAll(t, out, "No workspaces found.")
}

// Places two task files in a directory and runs them as a batch.
// Expects both tasks to be discovered and to complete successfully.
func TestCLI_Batch(t *testing.T) {
	dataDir := t.TempDir()
	t.Cleanup(func() { destroyAll(t, dataDir) })

	srcDir := t.TempDir()
	ensureRepo(t, srcDir)

	batchDir := t.TempDir()
	writeBatchTaskYAML(t, batchDir, "alpha", basicImage, srcDir)
	writeBatchTaskYAML(t, batchDir, "beta", basicImage, srcDir)

	out := runCLI(t, cliArgs(dataDir, "batch", batchDir)...)

	testutil.ContainsAll(t, out, "discovered 2 task(s)", "Batch ID:", "Summary: 2 total, 2 completed, 0 failed")
}

// Runs a task that produces commits inside the container, then uses
// "collect" to cherry-pick those commits into a target repository on a new branch.
// Verifies the branch exists.
func TestCLI_CollectAfterRun(t *testing.T) {
	dataDir := t.TempDir()
	t.Cleanup(func() { destroyAll(t, dataDir) })

	srcDir := t.TempDir()
	ensureRepo(t, srcDir)

	taskFile := writeTaskYAML(t, t.TempDir(), "collect-src", gitWorkImage, srcDir)
	out := runCLI(t, cliArgs(dataDir, "run", taskFile)...)
	id := parseID(t, out)

	targetRepo := t.TempDir()
	ensureRepo(t, targetRepo)

	out = runCLI(t, cliArgs(dataDir, "collect", "--repo", targetRepo, "--branch", "feat/test", id)...)
	testutil.ContainsAll(t, out, "Collected", "commit(s)")

	branchOut, err := exec.Command("git", "-C", targetRepo, "branch", "--list", "feat/test").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(branchOut), "feat/test") {
		t.Errorf("branch feat/test not found, got: %s", branchOut)
	}
}

// Runs a task that produces commits, then exports them as patch files.
// Verifies at least one .patch file is created.
func TestCLI_PatchesAfterRun(t *testing.T) {
	dataDir := t.TempDir()
	t.Cleanup(func() { destroyAll(t, dataDir) })

	srcDir := t.TempDir()
	ensureRepo(t, srcDir)

	taskFile := writeTaskYAML(t, t.TempDir(), "patches-src", gitWorkImage, srcDir)
	out := runCLI(t, cliArgs(dataDir, "run", taskFile)...)
	id := parseID(t, out)

	outDir := t.TempDir()
	out = runCLI(t, cliArgs(dataDir, "patches", "--dir", outDir, id)...)
	testutil.ContainsAll(t, out, "Exported", "patch(es)")

	patches, _ := filepath.Glob(filepath.Join(outDir, "*.patch"))
	if len(patches) == 0 {
		t.Fatalf("no .patch files in %s", outDir)
	}
}

func ensureRepo(t *testing.T, dir string) {
	t.Helper()
	if err := workspace.EnsureRepoReady(dir, "Test", "test@test.dev"); err != nil {
		t.Fatal(err)
	}
}
