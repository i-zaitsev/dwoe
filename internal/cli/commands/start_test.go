// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestStartCmd_Parse(t *testing.T) {
	t.Parallel()
	cmd := new(cmdStart)
	if err := cmd.Parse([]string{"ws-1"}); err != nil {
		t.Fatal(err)
	}
	if cmd.nameOrID != "ws-1" {
		t.Errorf("nameOrID = %q, want %q", cmd.nameOrID, "ws-1")
	}
}

func TestStartCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdStart))
}

func TestStartCmd_Run(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "pending", nil)
	cmd := &cmdStart{nameOrID: "ws-1"}

	err := cmd.Run(setup.env)

	if err != nil {
		t.Fatal(err)
	}
	testutil.ContainsAll(t, setup.stdout.String(), "Started workspace:", "Status: running")
}

func TestStartCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdStart{nameOrID: "no-such-ws"}

	err := cmd.Run(setup.env)

	testutil.WantErrAs[*state.NotFoundError](t, err)
}
