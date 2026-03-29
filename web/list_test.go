// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

func TestListWorkspaces(t *testing.T) {
	serv, source := newTestServer()
	for _, id := range []string{"ws-1", "ws-2", "ws-3"} {
		ws := workspace.New(state.EmptyWorkspace(id, "test-"+id), &config.Task{})
		source.Set(ws.ID, ws)
	}

	runHTTPTests(t, serv.handler, []httpCase{
		{"all", "/workspaces/list", 200, "test-ws-1"},
		{"filter", "/workspaces/list?q=ws-2", 200, "test-ws-2"},
		{"no_match", "/workspaces/list?q=nope", 200, "No workspaces"},
	})
}

func TestListWorkspaces_Batch(t *testing.T) {
	serv, source := newTestServer()
	source.Set("ws-1", workspace.New(state.EmptyWorkspace("ws-1", "test-ws-1"), &config.Task{}))
	source.Set("ws-2", workspace.New(state.EmptyWorkspace("ws-2", "test-ws-2"), &config.Task{}))
	source.Set("ws-3", workspace.New(state.EmptyWorkspace("ws-3", "test-ws-3"), &config.Task{}))
	source.batchRecordsFn = func() ([]*batch.Record, error) {
		return []*batch.Record{
			{
				ID:        "batch-abc",
				SourceDir: "/tmp/my-project",
				Entries: []batch.Entry{
					{WorkspaceID: "ws-1"},
					{WorkspaceID: "ws-2"},
				},
			},
		}, nil
	}

	runHTTPTests(t, serv.handler, []httpCase{
		{"batch_id", "/workspaces/list", 200, "batch-abc"},
		{"color", "/workspaces/list", 200, `data-batch="1"`},
		{"filter_batch", "/workspaces/list?batch=batch-abc", 200, "test-ws-1"},
		{"filter_excludes", "/workspaces/list?batch=batch-abc", 200, "test-ws-2"},
	})
}
