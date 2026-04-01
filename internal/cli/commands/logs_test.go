// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/state"
)

func TestLogsCmd_Parse(t *testing.T) {
	t.Parallel()
	cmd := new(cmdLogs)
	assert.NotErr(t, cmd.Parse([]string{"-f", "ws-test"}))
	assert.Equal(t, cmd.nameOrID, "ws-test")
	assert.Equal(t, cmd.follow, true)
}

func TestLogsCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	checkParseFails(t, new(cmdLogs))
}

func TestLogsCmd_Run_Follow(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)

	pr, pw := io.Pipe()
	setup.state.Data["ws-test"] = createWorkspace(t, t.TempDir(), "ws-test", "running", nil)
	setup.state.Data["ws-test"].ContainerIDs = map[string]string{"agent": fakeContainer}
	setup.docker.ContainerLogsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		return pr, nil
	}

	go func() {
		_, _ = fmt.Fprintf(pw, "line1\nline2\n")
		_, _ = fmt.Fprintf(pw, "line3\n")
		_ = pw.Close()
	}()

	cmd := &cmdLogs{nameOrID: "ws-test", follow: true}
	assert.NotErr(t, cmd.Run(setup.env))
	assert.Equal(t, setup.stdout.String(), "line1\nline2\nline3\n")
}

func TestLogsCmd_Run_SentinelInContent(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)

	setup.state.Data["ws-test"] = createWorkspace(t, t.TempDir(), "ws-test", "running", nil)
	setup.state.Data["ws-test"].ContainerIDs = map[string]string{"agent": fakeContainer}
	setup.docker.ContainerLogsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		return logReader(
			`before` + "\n" +
				`{"content":"Write exactly <promise>DONE</promise> as your final output"}` + "\n" +
				`after` + "\n",
		), nil
	}

	cmd := &cmdLogs{nameOrID: "ws-test", follow: true}
	assert.NotErr(t, cmd.Run(setup.env))
	assert.Equal(t, setup.stdout.String(),
		"before\n"+
			"{\"content\":\"Write exactly <promise>DONE</promise> as your final output\"}\n"+
			"after\n")
}

func TestLogsCmd_Run_NotFound(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	cmd := &cmdLogs{nameOrID: "no-such-ws", follow: true}

	err := cmd.Run(setup.env)

	assert.ErrAs[*state.NotFoundError](t, err)
}
