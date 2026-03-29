// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build integration

package cli_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func fatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func runCLI(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("%v\nstdout: %s\nstderr: %s", err, outBuf.String(), errBuf.String())
	}
	return outBuf.String()
}

func writeTaskYAML(t *testing.T, dir, name, image, localPath string) string {
	t.Helper()
	return testutil.WriteTaskFile(t, filepath.Join(dir, "task.yaml"), &config.Task{
		Name:    name,
		Source:  config.Source{LocalPath: localPath},
		Agent:   config.Agent{Image: image, Model: "test", MaxTurns: 1},
		Network: config.Network{Name: "clitest-" + name},
	})
}

func writeBatchTaskYAML(t *testing.T, dir, taskName, image, localPath string) string {
	t.Helper()
	return testutil.WriteBatchTaskFile(t, dir, &config.Task{
		Name:    taskName,
		Source:  config.Source{LocalPath: localPath},
		Agent:   config.Agent{Image: image, Model: "test", MaxTurns: 1},
		Network: config.Network{Name: "clitest-batch-" + taskName},
		Git:     config.GitUser{Name: "Test", Email: "test@test.dev"},
	})
}

var idPattern = regexp.MustCompile(`- ID:\s+(\S+)`)

func parseID(t *testing.T, output string) string {
	t.Helper()
	m := idPattern.FindStringSubmatch(output)
	if len(m) < 2 {
		t.Fatalf("cannot parse ID from output:\n%s", output)
	}
	return m[1]
}

func destroyAll(t *testing.T, dataDir string) {
	t.Helper()
	cmd := exec.Command(binaryPath, "--datadir", dataDir, "destroy", "--all", "--force")
	_ = cmd.Run()
}

func cliArgs(dataDir string, args ...string) []string {
	return append([]string{"--datadir", dataDir, "--loglevel", "error"}, args...)
}
