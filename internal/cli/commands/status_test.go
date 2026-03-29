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

func TestStatusCmd_Parse(t *testing.T) {
	t.Parallel()
	cmd := new(cmdStatus)
	if err := cmd.Parse([]string{"ws-1"}); err != nil {
		t.Fatal(err)
	}
	if cmd.nameOrID != "ws-1" {
		t.Errorf("nameOrID = %q, want %q", cmd.nameOrID, "ws-1")
	}
}

func TestStatusCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdStatus))
}

func TestStatusCmd_Run(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	now := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "running", &now)
	cmd := &cmdStatus{nameOrID: "ws-1"}

	err := cmd.Run(setup.env)

	if err != nil {
		t.Fatal(err)
	}
	testutil.ContainsAll(t, setup.stdout.String(),
		"Name:     ws-1 name",
		"ID:       ws-1",
		"Status:   running",
		"Created:  2001-01-01 00:00:00",
	)
}

func TestStatusCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdStatus{nameOrID: "no-such-ws"}

	err := cmd.Run(setup.env)

	testutil.WantErrAs[*state.NotFoundError](t, err)
}
