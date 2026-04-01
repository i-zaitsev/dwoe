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

func TestDestroyCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		args      []string
		wantID    string
		wantForce bool
		wantAll   bool
	}{
		{name: "positional", args: []string{"ws-1"}, wantID: "ws-1"},
		{name: "force_short", args: []string{"-f", "ws-1"}, wantID: "ws-1", wantForce: true},
		{name: "all_force", args: []string{"--all", "--force"}, wantForce: true, wantAll: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := new(cmdDestroy)
			assert.NotErr(t, cmd.Parse(tc.args))
			assert.Equal(t, cmd.nameOrID, tc.wantID)
			assert.Equal(t, cmd.force, tc.wantForce)
			assert.Equal(t, cmd.all, tc.wantAll)
		})
	}
}

func TestDestroyCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdDestroy))
}

func TestDestroyCmd_Run_Stopped(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "stopped", nil)
	setup.state.Data["ws-1"].ContainerIDs = map[string]string{"agent": fakeContainer}
	setup.state.Data["ws-1"].NetworkID = "fake-net"
	cmd := &cmdDestroy{nameOrID: "ws-1"}

	assert.NotErr(t, cmd.Run(setup.env))
	assert.ContainsAll(t, setup.stdout.String(), "Destroying workspace:", "Workspace destroyed.")
	if _, ok := setup.state.Data["ws-1"]; ok {
		t.Error("workspace should be deleted from state")
	}
}

func TestDestroyCmd_Run_RunningRequiresForce(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "running", nil)
	setup.state.Data["ws-1"].ContainerIDs = map[string]string{"agent": fakeContainer}
	cmd := &cmdDestroy{nameOrID: "ws-1"}

	err := cmd.Run(setup.env)

	assert.ErrIs(t, err, errWsRunning)
}

func TestDestroyCmd_Run_RunningWithForce(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "running", nil)
	setup.state.Data["ws-1"].ContainerIDs = map[string]string{"agent": fakeContainer}
	setup.state.Data["ws-1"].NetworkID = "fake-net"
	cmd := &cmdDestroy{nameOrID: "ws-1", force: true}

	assert.NotErr(t, cmd.Run(setup.env))
	if _, ok := setup.state.Data["ws-1"]; ok {
		t.Error("workspace should be deleted from state")
	}
}

func TestDestroyCmd_Run_All(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	dir := t.TempDir()
	setup.state.Data["ws-1"] = createWorkspace(t, dir, "ws-1", "stopped", nil)
	setup.state.Data["ws-2"] = createWorkspace(t, dir, "ws-2", "running", nil)
	setup.state.Data["ws-2"].ContainerIDs = map[string]string{"agent": fakeContainer}
	cmd := &cmdDestroy{all: true, force: true}

	assert.NotErr(t, cmd.Run(setup.env))
	assert.Zero(t, len(setup.state.Data))
}

func TestDestroyCmd_Run_AllWithoutForce(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdDestroy{all: true}

	err := cmd.Run(setup.env)

	assert.ErrAs[*cli.ArgMissingError](t, err)
}

func TestDestroyCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdDestroy{nameOrID: "no-such-ws"}

	err := cmd.Run(setup.env)

	assert.ErrAs[*state.NotFoundError](t, err)
}
