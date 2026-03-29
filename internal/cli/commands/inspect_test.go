// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"testing"
	"time"

	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestInspectCmd_Parse(t *testing.T) {
	t.Parallel()
	cmd := new(cmdInspect)
	if err := cmd.Parse([]string{"ws-1"}); err != nil {
		t.Fatal(err)
	}
	if cmd.nameOrID != "ws-1" {
		t.Errorf("nameOrID = %q, want %q", cmd.nameOrID, "ws-1")
	}
}

func TestInspectCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdInspect))
}

func TestInspectCmd_Run(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	now := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	ws := createWorkspace(t, t.TempDir(), "ws-1", "running", &now)
	ws.ContainerIDs = map[string]string{"agent": "ctr-agent-1", "proxy": "ctr-proxy-1"}
	ws.NetworkID = "net-abc"
	setup.state.Data["ws-1"] = ws
	cmd := &cmdInspect{nameOrID: "ws-1"}

	err := cmd.Run(setup.env)

	if err != nil {
		t.Fatal(err)
	}
	testutil.ContainsAll(t, setup.stdout.String(),
		"Name:       ws-1 name",
		"ID:         ws-1",
		"Status:     running",
		"BasePath:",
		"Agent:      ctr-agent-1",
		"Proxy:      ctr-proxy-1",
		"NetworkID:  net-abc",
		"Created:    2001-01-01 00:00:00",
	)
}

func TestInspectCmd_Run_NilContainerIDs(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	now := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	ws := createWorkspace(t, t.TempDir(), "ws-cleanup", "completed", &now)
	ws.ContainerIDs = nil
	setup.state.Data["ws-cleanup"] = ws
	cmd := &cmdInspect{nameOrID: "ws-cleanup"}

	err := cmd.Run(setup.env)

	if err != nil {
		t.Fatal(err)
	}
	testutil.ContainsAll(t, setup.stdout.String(),
		"Agent:      (none)",
		"Proxy:      (none)",
	)
}

func TestInspectCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdInspect{nameOrID: "no-such-ws"}

	err := cmd.Run(setup.env)

	testutil.WantErrAs[*state.NotFoundError](t, err)
}

func TestInspectCmd_Run_AmbiguousPrefix(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	now := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	setup.state.Data["abc-111"] = createWorkspace(t, dir, "abc-111", "running", &now)
	setup.state.Data["abc-222"] = createWorkspace(t, dir, "abc-222", "stopped", &now)
	cmd := &cmdInspect{nameOrID: "abc"}

	err := cmd.Run(setup.env)

	testutil.WantErrAs[*state.AmbiguousMatchError](t, err)
}

func TestInspectCmd_Run_UniquePrefix(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	now := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	setup.state.Data["abc-111"] = createWorkspace(t, dir, "abc-111", "running", &now)
	setup.state.Data["xyz-999"] = createWorkspace(t, dir, "xyz-999", "stopped", &now)
	cmd := &cmdInspect{nameOrID: "abc"}

	err := cmd.Run(setup.env)

	if err != nil {
		t.Fatal(err)
	}
	testutil.ContainsAll(t, setup.stdout.String(),
		"ID:         abc-111",
		"Status:     running",
	)
}
