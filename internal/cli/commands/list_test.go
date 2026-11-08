// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/i-zaitsev/dwoe/internal/workspace"
	"github.com/i-zaitsev/pkg/assert"
	"github.com/i-zaitsev/pkg/cmd"
)

func TestListCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		args       []string
		wantFormat string
	}{
		{name: "default", args: nil, wantFormat: "table"},
		{name: "table", args: []string{"-format", "table"}, wantFormat: "table"},
		{name: "json", args: []string{"-format", "json"}, wantFormat: "json"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := new(cmdList)
			assert.ErrVal(t, cmd.Parse(tc.args)).Nil().Fatal()
			assert.Eq(t, cmd.format, tc.wantFormat)
		})
	}
}

func TestListCmd_Parse_InvalidFormat(t *testing.T) {
	t.Parallel()
	c := new(cmdList)
	err := c.Parse([]string{"-format", "xml"})
	assert.ErrVal(t, err).As(new(*cmd.ArgInvalidError)).Fatal()
}

func TestListCmd_Run_Table(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		format string
	}{
		{name: "default_table", format: "table"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			setup := newListTestSetup(t, false)
			cmd := &cmdList{format: tc.format}

			assert.ErrVal(t, cmd.Run(setup.env)).Nil().Fatal()
			want := strings.Join([]string{
				"NAME       ID    STATUS   EXIT         CREATED              STARTED              FINISHED",
				"ws-1 name  ws-1  running  pending      2001-01-01 00:00:00                       ",
				"ws-2 name  ws-2  stopped  pending      2001-01-01 00:00:00  2001-01-01 00:00:00  2001-01-01 01:00:00",
				"ws-3 name  ws-3  failed   exit code 1  2001-01-01 00:00:00  2001-01-01 01:00:00  2001-01-02 00:00:00",
				"",
			}, "\n")
			if diff := cmp.Diff(want, setup.stdout.String()); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestListCmd_Run_JSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		want  int
		empty bool
	}{
		{want: 3, empty: false},
		{want: 0, empty: true},
	}

	for _, tc := range testCases {

		t.Run(fmt.Sprintf("empty=%t", tc.empty), func(t *testing.T) {
			t.Parallel()

			setup := newListTestSetup(t, tc.empty)
			cmd := &cmdList{format: "json"}
			assert.ErrVal(t, cmd.Run(setup.env)).Nil().Fatal()

			var wss []*workspace.Workspace
			buf := strings.NewReader(setup.stdout.String())
			dec := json.NewDecoder(buf)
			assert.ErrVal(t, dec.Decode(&wss)).Nil().Fatal()
			assert.Eq(t, len(wss), tc.want)
		})
	}
}

func TestListCmd_Run_TableEmpty(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdList{format: "table"}

	err := cmd.Run(setup.env)

	assert.ErrVal(t, err).Nil().Fatal()
	assert.Contains(t, setup.stdout.String(), "No workspaces found")
}

func newListTestSetup(t *testing.T, empty bool) *cmdTestSetup {
	t.Helper()

	setup := newCmdTestSetup(t)

	if empty {
		return setup
	}

	now := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	plusHour := now.Add(time.Hour)
	plusDay := now.Add(24 * time.Hour)

	dir := t.TempDir()

	setup.state.Data["ws-1"] = createWorkspace(t, dir, "ws-1", "running", &now)

	setup.state.Data["ws-2"] = createWorkspace(t, dir, "ws-2", "stopped", &now)
	setup.state.Data["ws-2"].StartedAt = &now
	setup.state.Data["ws-2"].FinishedAt = &plusHour

	exitCode := 1
	setup.state.Data["ws-3"] = createWorkspace(t, dir, "ws-3", "failed", &now)
	setup.state.Data["ws-3"].StartedAt = &plusHour
	setup.state.Data["ws-3"].FinishedAt = &plusDay
	setup.state.Data["ws-3"].ExitCode = &exitCode

	return setup
}
