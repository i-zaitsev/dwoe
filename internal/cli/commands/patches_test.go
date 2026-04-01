// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/state"
)

func TestPatchesCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		wantID  string
		wantDir string
	}{
		{
			name:    "single_workspace",
			args:    []string{"--dir", "/tmp/out", "ws-1"},
			wantID:  "ws-1",
			wantDir: "/tmp/out",
		},
		{
			name:    "dir_last",
			args:    []string{"--dir", "/patches", "my-ws"},
			wantID:  "my-ws",
			wantDir: "/patches",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := new(cmdPatches)
			assert.NotErr(t, cmd.Parse(tc.args))
			assert.Equal(t, cmd.nameOrID, tc.wantID)
			assert.Equal(t, cmd.dir, tc.wantDir)
		})
	}
}

func TestPatchesCmd_Parse_Batch(t *testing.T) {
	t.Parallel()
	cmd := new(cmdPatches)
	assert.NotErr(t, cmd.Parse([]string{"--batch", "abc-123", "--dir", "/tmp/out"}))
	assert.Equal(t, cmd.batchID, "abc-123")
	assert.Equal(t, cmd.dir, "/tmp/out")
}

func TestPatchesCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdPatches))
}

func TestPatchesCmd_Parse_MissingDir(t *testing.T) {
	t.Parallel()
	assert.ErrAs[*cli.ArgMissingError](t, new(cmdPatches).Parse([]string{"ws-1"}))
}

func TestPatchesCmd_Parse_MissingNameOrID(t *testing.T) {
	t.Parallel()
	assert.ErrAs[*cli.ArgMissingError](t, new(cmdPatches).Parse([]string{"--dir", "/tmp"}))
}

func TestPatchesCmd_Run_Single(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "completed", nil)
	cmd := &cmdPatches{
		nameOrID: "ws-1", dir: t.TempDir(),
		exportPatch: func(_, _ string) (int, error) { return 5, nil },
	}

	err := cmd.Run(setup.env)

	assert.NotErr(t, err)
	assert.ContainsAll(t, setup.stdout.String(), "Exported 5 patch(es)")
}

func TestPatchesCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdPatches{nameOrID: "no-such", dir: t.TempDir()}

	err := cmd.Run(setup.env)

	assert.ErrAs[*state.NotFoundError](t, err)
}

func TestPatchesCmd_Run_Running(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "running", nil)
	cmd := &cmdPatches{nameOrID: "ws-1", dir: t.TempDir()}

	assert.Err(t, cmd.Run(setup.env))
}
