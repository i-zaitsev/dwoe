// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/state"
)

func TestStopCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		args      []string
		wantID    string
		wantForce bool
	}{
		{name: "positional", args: []string{"ws-1"}, wantID: "ws-1"},
		{name: "force_short", args: []string{"-f", "ws-1"}, wantID: "ws-1", wantForce: true},
		{name: "force_long", args: []string{"-force", "ws-1"}, wantID: "ws-1", wantForce: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := new(cmdStop)
			assert.NotErr(t, cmd.Parse(tc.args))
			assert.Equal(t, cmd.nameOrID, tc.wantID)
			assert.Equal(t, cmd.force, tc.wantForce)
		})
	}
}

func TestStopCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdStop))
}

func TestStopCmd_Run(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "running", nil)
	setup.state.Data["ws-1"].ContainerIDs = map[string]string{"agent": fakeContainer}
	cmd := &cmdStop{nameOrID: "ws-1"}

	assert.NotErr(t, cmd.Run(setup.env))
	assert.ContainsAll(t, setup.stdout.String(), "Stopping workspace:", "Workspace stopped.")
}

func TestStopCmd_Run_Force(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "running", nil)
	setup.state.Data["ws-1"].ContainerIDs = map[string]string{"agent": fakeContainer}
	cmd := &cmdStop{nameOrID: "ws-1", force: true}

	assert.NotErr(t, cmd.Run(setup.env))
	assert.ContainsAll(t, setup.stdout.String(), "Workspace stopped.")
}

func TestStopCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdStop{nameOrID: "no-such-ws"}

	err := cmd.Run(setup.env)

	assert.ErrAs[*state.NotFoundError](t, err)
}
