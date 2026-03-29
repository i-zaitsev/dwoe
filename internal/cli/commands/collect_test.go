// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/testutil"
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
			if err := cmd.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			if cmd.nameOrID != tc.wantID {
				t.Errorf("nameOrID = %q, want %q", cmd.nameOrID, tc.wantID)
			}
			if cmd.repo != tc.wantRepo {
				t.Errorf("repo = %q, want %q", cmd.repo, tc.wantRepo)
			}
			if cmd.branch != tc.wantBranch {
				t.Errorf("branch = %q, want %q", cmd.branch, tc.wantBranch)
			}
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
			testutil.WantErrAs[*cli.ArgMissingError](t, new(cmdCollect).Parse(tc.args))
		})
	}
}

func TestCollectCmd_Parse_Batch(t *testing.T) {
	t.Parallel()
	cmd := new(cmdCollect)
	if err := cmd.Parse([]string{"--batch", "abc-123"}); err != nil {
		t.Fatal(err)
	}
	if cmd.batchID != "abc-123" {
		t.Errorf("batchID = %q, want %q", cmd.batchID, "abc-123")
	}
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

	if err != nil {
		t.Fatal(err)
	}
	testutil.ContainsAll(t, setup.stdout.String(), "Collected 3 commit(s)")
}

func TestCollectCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdCollect{nameOrID: "no-such", repo: "/tmp", branch: "b"}

	err := cmd.Run(setup.env)

	testutil.WantErrAs[*state.NotFoundError](t, err)
}

func TestCollectCmd_Run_Running(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "running", nil)
	cmd := &cmdCollect{nameOrID: "ws-1", repo: "/tmp", branch: "b"}

	err := cmd.Run(setup.env)

	if err == nil {
		t.Fatal("expected error for running workspace")
	}
}

func TestCollectCmd_Run_Pending(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", "pending", nil)
	cmd := &cmdCollect{nameOrID: "ws-1", repo: "/tmp", branch: "b"}

	err := cmd.Run(setup.env)

	if err == nil {
		t.Fatal("expected error for pending workspace")
	}
}
