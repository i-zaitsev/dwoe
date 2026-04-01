// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/state"
)

func TestStartCmd_Parse(t *testing.T) {
	t.Parallel()
	cmd := new(cmdStart)
	assert.NotErr(t, cmd.Parse([]string{"ws-1"}))
	assert.Equal(t, cmd.nameOrID, "ws-1")
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

	assert.NotErr(t, cmd.Run(setup.env))
	assert.ContainsAll(t, setup.stdout.String(), "Started workspace:", "Status: running")
}

func TestStartCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdStart{nameOrID: "no-such-ws"}

	err := cmd.Run(setup.env)

	assert.ErrAs[*state.NotFoundError](t, err)
}
