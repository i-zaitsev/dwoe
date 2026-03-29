// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"os"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestCreateCmd_Parse(t *testing.T) {
	t.Parallel()
	cmd := new(cmdCreate)
	err := cmd.Parse([]string{"-name", "custom", "/path/to/task.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.taskPath != "/path/to/task.yaml" {
		t.Errorf("taskPath = %q, want %q", cmd.taskPath, "/path/to/task.yaml")
	}
	if cmd.name != "custom" {
		t.Errorf("name = %q, want %q", cmd.name, "custom")
	}
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

	if err != nil {
		t.Fatal(err)
	}

	nDirs := testutil.DirCount(ts.env.DataDir(), "workspaces", "*", "workspace")
	if nDirs != 1 {
		t.Errorf("dir count = %d, want 1", nDirs)
	}

	testutil.ContainsAll(t, ts.stdout.String(), "Created workspace: test-task", "Status: pending")
}

func TestCreateCmd_Run_MissingFile(t *testing.T) {
	t.Parallel()
	ts := newCmdTestSetup(t)
	cmd := &cmdCreate{taskPath: "/no/such/file.yaml"}

	err := cmd.Run(ts.env)

	testutil.WantErr(t, err, os.ErrNotExist)
}
