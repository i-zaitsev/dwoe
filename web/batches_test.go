// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"testing"
	"time"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

func TestBatchesList(t *testing.T) {
	serv, source := newTestServer()

	now := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	for _, id := range []string{"ws-1", "ws-2", "ws-3"} {
		sw := state.EmptyWorkspace(id, "test-"+id)
		sw.StartedAt = &now
		source.Set(id, workspace.New(sw, &config.Task{}))
	}

	source.batchRecordsFn = func() ([]*batch.Record, error) {
		return []*batch.Record{
			{
				ID: "batch-abcdef12",
				Entries: []batch.Entry{
					{WorkspaceID: "ws-1"},
					{WorkspaceID: "ws-2"},
					{WorkspaceID: "ws-3"},
				},
			},
		}, nil
	}

	runHTTPTests(t, serv.handler, []httpCase{
		{"has_batch_id", "/batches", 200, "batch-ab"},
		{"has_workers", "/batches", 200, "3"},
	})
}

func TestBatchesList_empty(t *testing.T) {
	serv, _ := newTestServer()

	runHTTPTests(t, serv.handler, []httpCase{
		{"no_batches", "/batches", 200, "No batches"},
	})
}

func TestBatchesChanges(t *testing.T) {
	serv, source := newTestServer()

	source.Set("ws-1", workspace.New(state.EmptyWorkspace("ws-1", "alpha"), &config.Task{}))
	source.Set("ws-2", workspace.New(state.EmptyWorkspace("ws-2", "beta"), &config.Task{}))

	source.batchRecordsFn = func() ([]*batch.Record, error) {
		return []*batch.Record{
			{
				ID: "batch-abc",
				Entries: []batch.Entry{
					{WorkspaceID: "ws-1"},
					{WorkspaceID: "ws-2"},
				},
			},
		}, nil
	}

	source.diffFn = func(id string) (*workspace.DiffInfo, error) {
		switch id {
		case "ws-1":
			return &workspace.DiffInfo{
				Commits: []workspace.CommitInfo{{Hash: "aaa1111", Message: "feat alpha"}},
				Stat:    " alpha.go | 1 +",
				Diff:    "+package alpha",
			}, nil
		case "ws-2":
			return &workspace.DiffInfo{
				Commits: []workspace.CommitInfo{{Hash: "bbb2222", Message: "feat beta"}},
				Stat:    " beta.go | 1 +",
				Diff:    "+package beta",
			}, nil
		}
		return nil, &state.NotFoundError{ID: id}
	}

	runHTTPTests(t, serv.handler, []httpCase{
		{"has_commit", "/batches?batch=batch-abc", 200, "aaa1111"},
		{"has_name", "/batches?batch=batch-abc", 200, "alpha"},
		{"has_second", "/batches?batch=batch-abc", 200, "bbb2222"},
		{"has_diff", "/batches?batch=batch-abc", 200, "package alpha"},
	})
}

func TestBatchesChanges_missing(t *testing.T) {
	serv, _ := newTestServer()

	runHTTPTests(t, serv.handler, []httpCase{
		{"no_changes", "/batches?batch=nope", 200, "No changes"},
	})
}
