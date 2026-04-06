// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"os"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/testutil"
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

func TestCreateCmd_Run_MissingFile(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	cmd := &cmdCreate{taskPath: "/no/such/file.yaml"}

	err := cmd.Run(ts.env)

	assert.ErrIs(t, err, os.ErrNotExist)
}
