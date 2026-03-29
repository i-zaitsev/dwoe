// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

func TestWorkspaceLogs(t *testing.T) {
	serv, source := newTestServer()
	ws := workspace.New(state.EmptyWorkspace("ws-1", "test-ws"), &config.Task{})
	source.Set(ws.ID, ws)
	source.logsFn = func(_ context.Context, id string, _ bool) (io.ReadCloser, error) {
		if id == ws.ID {
			return io.NopCloser(strings.NewReader("line one\nline two\nline three\n")), nil
		}
		return nil, &state.NotFoundError{ID: id}
	}

	runHTTPTests(t, serv.handler, []httpCase{
		{"logs", "/workspaces/logs?q=ws-1", 200, "line one"},
		{"by_name", "/workspaces/logs?q=test-ws", 200, "line three"},
		{"not_found", "/workspaces/logs?q=nope", 404, "not found"},
		{"missing_param", "/workspaces/logs", 400, "required"},
	})
}

func TestRenderLogLine_PlainText(t *testing.T) {
	got := renderLogLine("just a plain line")
	assert.Contains(t, got, `class="log-line"`)
	assert.Contains(t, got, "just a plain line")
}

func TestRenderLogLine_JSON(t *testing.T) {
	input := `{"key":"value","num":1}`
	got := renderLogLine(input)

	assert.Contains(t, got, `class="jkey"`)
	assert.Contains(t, got, `class="js"`)
	assert.Contains(t, got, `class="jn"`)
	assert.Contains(t, got, `<details`)
	assert.Contains(t, got, `<summary>`)
}

func TestLogsView(t *testing.T) {
	serv, source := newTestServer()
	ws := workspace.New(state.EmptyWorkspace("ws-1", "test-ws"), &config.Task{})
	source.Set(ws.ID, ws)

	runHTTPTests(t, serv.handler, []httpCase{
		{"by_id", "/workspaces/logs/view?q=ws-1", 200, "test-ws"},
		{"by_name", "/workspaces/logs/view?q=test-ws", 200, "ws-1"},
		{"not_found", "/workspaces/logs/view?q=nope", 404, "not found"},
		{"missing_param", "/workspaces/logs/view", 400, "required"},
	})
}

func TestWorkspaceLogs_Follow(t *testing.T) {
	source := newFakeSource()

	pr, pw := io.Pipe()
	source.logsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		return pr, nil
	}

	go func() {
		_, _ = fmt.Fprintf(pw, "line1\nline2\n")
		_, _ = fmt.Fprintf(pw, "line3\n")
		pw.Close()
	}()

	rc, err := source.Logs(context.Background(), "any", true)
	assert.NotErr(t, err)

	var lines []string
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	rc.Close()

	assert.Equal(t, len(lines), 3)
	assert.Equal(t, lines[0], "line1")
	assert.Equal(t, lines[2], "line3")
}
