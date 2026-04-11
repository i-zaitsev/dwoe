// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/sentinel"
	"github.com/i-zaitsev/dwoe/internal/testfake"
	"github.com/i-zaitsev/dwoe/internal/testutil"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

func TestCreateCmd_Parse(t *testing.T) {
	t.Parallel()
	cmd := new(cmdCreate)
	assert.NotErr(t, cmd.Parse([]string{"-name", "custom", "/path/to/task.yaml"}))
	assert.Equal(t, cmd.taskPath, "/path/to/task.yaml")
	assert.Equal(t, cmd.name, "custom")
}

func TestCreateCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdCreate))
}

func TestCreateCmd_Run(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	cmd := &cmdCreate{taskPath: writeTaskFile(t, t.TempDir(), "test-task")}

	err := cmd.Run(ts.env)

	assert.NotErr(t, err)
	nDirs := testutil.DirCount(ts.env.DataDir(), "workspaces", "*", "workspace")
	assert.Equal(t, nDirs, 1)
	assert.ContainsAll(t, ts.stdout.String(), "Workspace created: test-task", "Status: pending")
}

func TestCreateCmd_Run_SkipIfDone(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)

	srcDir := t.TempDir()
	dir := t.TempDir()
	ws := testfake.CreateWorkspace(t, dir, "ws-done", "done-task", "completed")
	setup.state.Data["ws-done"] = ws

	taskYAML := fmt.Sprintf(
		"name: done-task\ncontinue_policy: resume\nsource:\n  local_path: %s\n", srcDir,
	)
	testutil.WriteFile(t, filepath.Join(ws.BasePath, "config.yaml"), taskYAML)

	sen := sentinel.FromConfig(&config.Task{
		Name:   "done-task",
		Source: config.Source{LocalPath: srcDir},
	})
	assert.NotErr(t, sen.Write(ws.BasePath))

	taskFile := filepath.Join(t.TempDir(), "task.yaml")
	testutil.WriteFile(t, taskFile, taskYAML)

	cmd := &cmdCreate{taskPath: taskFile}
	err := cmd.Run(setup.env)

	assert.ErrIs(t, err, workspace.ErrWorkspaceDone)
	assert.Contains(t, setup.stdout.String(), "Workspace already done")
}

func TestCreateCmd_Run_MissingFile(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	cmd := &cmdCreate{taskPath: "/no/such/file.yaml"}

	err := cmd.Run(ts.env)

	assert.ErrIs(t, err, os.ErrNotExist)
}
