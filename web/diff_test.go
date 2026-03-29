// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

func TestFormatDiff(t *testing.T) {
	raw := "+added\n-removed\n@@ -1,3 +1,5 @@\n context"
	got := string(formatDiff(raw))
	assert.Contains(t, got, `<span class="da">+added</span>`)
	assert.Contains(t, got, `<span class="dd">-removed</span>`)
	assert.Contains(t, got, `<span class="dh">@@ -1,3 +1,5 @@</span>`)
	assert.Contains(t, got, "context")
}

func TestDiffWorkspace(t *testing.T) {
	serv, source := newTestServer()
	ws := workspace.New(state.EmptyWorkspace("ws-1", "test-ws"), &config.Task{})
	source.Set(ws.ID, ws)
	source.diffFn = func(id string) (*workspace.DiffInfo, error) {
		if id == "ws-1" {
			return &workspace.DiffInfo{
				Commits: []workspace.CommitInfo{
					{Hash: "abc1234", Message: "add feature"},
				},
				Stat: " feature.go | 1 +\n 1 file changed",
				Diff: "+package main",
			}, nil
		}
		return nil, &state.NotFoundError{ID: id}
	}

	runHTTPTests(t, serv.handler, []httpCase{
		{"valid", "/workspaces/diff?q=ws-1", 200, "abc1234"},
		{"content", "/workspaces/diff?q=ws-1", 200, "add feature"},
		{"stat", "/workspaces/diff?q=ws-1", 200, "feature.go"},
		{"not_found", "/workspaces/diff?q=nope", 404, "not found"},
		{"missing_param", "/workspaces/diff", 400, "required"},
	})
}
