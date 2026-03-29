// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/testfake"
	"github.com/i-zaitsev/dwoe/internal/testutil"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

type cmdTestSetup struct {
	env    *cli.Env
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	docker *testfake.FakeDocker
	state  *testfake.FakeState
}

func newCmdTestSetup(t *testing.T) *cmdTestSetup {
	t.Helper()
	var stdout, stderr bytes.Buffer
	e := cli.NewEnv(&stdout, &stderr)
	fd := new(testfake.FakeDocker)
	fs := testfake.NewFakeState()
	dataDir := t.TempDir()
	e.SetDataDir(dataDir)
	e.SetNewManager(func() (*workspace.Manager, error) {
		return workspace.NewManagerWith(dataDir, fd, fs)
	})
	return &cmdTestSetup{
		env:    e,
		stdout: &stdout,
		stderr: &stderr,
		docker: fd,
		state:  fs,
	}
}

const fakeContainer = "fake-container"

func writeTaskFile(t *testing.T, dir, name string) string {
	t.Helper()
	return testutil.WriteTaskFile(t, filepath.Join(dir, "task.yaml"), &config.Task{
		Name:   name,
		Source: config.Source{LocalPath: t.TempDir()},
	})
}

func writeBatchTaskFile(t *testing.T, dir, taskName, srcDir string) string {
	t.Helper()
	return testutil.WriteBatchTaskFile(t, dir, &config.Task{
		Name:   taskName,
		Source: config.Source{LocalPath: srcDir},
	})
}

func logReader(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func checkParseFails(t *testing.T, cmd cli.Command) {
	t.Helper()
	t.Run("nil_args", func(t *testing.T) {
		t.Parallel()
		testutil.WantErrAs[*cli.ArgMissingError](t, cmd.Parse(nil))
	})
	t.Run("empty_args", func(t *testing.T) {
		t.Parallel()
		testutil.WantErrAs[*cli.ArgMissingError](t, cmd.Parse([]string{}))
	})
	t.Run("bad_flag", func(t *testing.T) {
		t.Parallel()
		testutil.WantErrAs[*cli.FlagParseError](t, cmd.Parse([]string{"-badflag"}))
	})
}

func createWorkspace(t *testing.T, dir, id, status string, created *time.Time) *state.Workspace {
	t.Helper()
	ws := testfake.CreateWorkspace(t, dir, id, id+" name", status)
	ws.CreatedAt = created
	return ws
}
