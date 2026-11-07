// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/pkg/assert"
)

func TestCollectCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		args       []string
		wantID     string
		wantRepo   string
		wantBranch string
	}{
		{
			name:       "all_flags",
			args:       []string{"--repo", "/tmp/repo", "--branch", "feat/x", "ws-1"},
			wantID:     "ws-1",
			wantRepo:   "/tmp/repo",
			wantBranch: "feat/x",
		},
		{
			name:       "positional_last",
			args:       []string{"--repo", "/other", "--branch", "b", "my-ws"},
			wantID:     "my-ws",
			wantRepo:   "/other",
			wantBranch: "b",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := new(cmdCollect)

			assert.ErrVal(t, cmd.Parse(tc.args)).Nil().Fatal()

			assert.Eq(t, cmd.nameOrID, tc.wantID)
			assert.Eq(t, cmd.repo, tc.wantRepo)
			assert.Eq(t, cmd.branch, tc.wantBranch)
		})
	}
}

func TestCollectCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdCollect))
}

func TestCollectCmd_Parse_MissingFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{"missing_repo", []string{"--branch", "b", "ws-1"}},
		{"missing_branch", []string{"--repo", "/r", "ws-1"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrVal(t, new(cmdCollect).Parse(tc.args)).As(new(*cli.ArgMissingError)).Fatal()
		})
	}
}

func TestCollectCmd_Parse_Batch(t *testing.T) {
	t.Parallel()
	cmd := new(cmdCollect)
	assert.ErrVal(t, cmd.Parse([]string{"--batch", "abc-123"})).Nil().Fatal()
	assert.Eq(t, cmd.batchID, "abc-123")
}

func TestCollectCmd_Run_Single(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "completed", nil)
	cmd := &cmdCollect{
		nameOrID: "ws-1", repo: "/tmp/repo", branch: "feat/x",
		collect: func(_, _, _ string) (int, error) { return 3, nil },
	}

	err := cmd.Run(setup.env)

	assert.ErrVal(t, err).Nil().Fatal()
	assert.ContainsAll(t, setup.stdout.String(), "Collected 3 commit(s)")
}

func TestCollectCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdCollect{nameOrID: "no-such", repo: "/tmp", branch: "b"}

	err := cmd.Run(setup.env)

	assert.ErrVal(t, err).As(new(*state.NotFoundError)).Fatal()
}

func TestCollectCmd_Run_Running(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "running", nil)
	cmd := &cmdCollect{nameOrID: "ws-1", repo: "/tmp", branch: "b"}

	err := cmd.Run(setup.env)

	assert.ErrVal(t, err).NotNil().Fatal()
}

func TestCollectCmd_Run_Pending(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "pending", nil)
	cmd := &cmdCollect{nameOrID: "ws-1", repo: "/tmp", branch: "b"}

	err := cmd.Run(setup.env)

	assert.ErrVal(t, err).NotNil().Fatal()
}
