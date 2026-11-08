// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"errors"
	"testing"
	"time"

	"github.com/i-zaitsev/pkg/assert"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

var (
	t0 = time.Date(2025, time.March, 1, 10, 0, 0, 0, time.UTC)
	t1 = t0.Add(time.Hour)
	t2 = t0.Add(2 * time.Hour)
)

func batchWorkspace(id string, started, finished *time.Time) *workspace.Workspace {
	sw := state.EmptyWorkspace(id, "name-"+id)
	sw.StartedAt = started
	sw.FinishedAt = finished
	return workspace.New(sw, &config.Task{})
}

// lookupIn resolves workspaces from a map, like the manager does from its store.
func lookupIn(known map[string]*workspace.Workspace) lookupFunc {
	return func(id string) (*workspace.Workspace, error) {
		if ws, ok := known[id]; ok {
			return ws, nil
		}
		return nil, &state.NotFoundError{ID: id}
	}
}

func record(id string, workspaceIDs ...string) *batch.Record {
	rec := &batch.Record{ID: id}
	for _, wsID := range workspaceIDs {
		rec.Entries = append(rec.Entries, batch.Entry{WorkspaceID: wsID})
	}
	return rec
}

func TestBatchIndex(t *testing.T) {
	records := []*batch.Record{
		record("b1", "ws-1", "ws-2"),
		record("b2", "ws-3"),
		record("b3"), record("b4"),
		record("b5", "ws-5"),
	}

	index := batchIndex(records)

	assert.Eq(t, index["ws-1"], batchRef{ID: "b1", Color: 1})
	assert.Eq(t, index["ws-2"], batchRef{ID: "b1", Color: 1})
	assert.Eq(t, index["ws-3"], batchRef{ID: "b2", Color: 2})
	assert.Eq(t, index["ws-5"], batchRef{ID: "b5", Color: 1})
	assert.Eq(t, index["unknown"], batchRef{})
}

func TestSummarizeBatch(t *testing.T) {
	tests := []struct {
		name         string
		known        map[string]*workspace.Workspace
		wantWorkers  int
		wantStarted  *time.Time
		wantFinished *time.Time
	}{
		{
			name: "all_finished",
			known: map[string]*workspace.Workspace{
				"ws-1": batchWorkspace("ws-1", &t1, &t2),
				"ws-2": batchWorkspace("ws-2", &t0, &t1),
			},
			wantWorkers: 2, wantStarted: &t0, wantFinished: &t2,
		},
		{
			name: "one_running_keeps_finish_empty",
			known: map[string]*workspace.Workspace{
				"ws-1": batchWorkspace("ws-1", &t0, &t1),
				"ws-2": batchWorkspace("ws-2", &t1, nil),
			},
			wantWorkers: 2, wantStarted: &t0, wantFinished: nil,
		},
		{
			name: "unresolved_workspaces_are_not_workers",
			known: map[string]*workspace.Workspace{
				"ws-1": batchWorkspace("ws-1", &t0, &t1),
			},
			wantWorkers: 1, wantStarted: &t0, wantFinished: &t1,
		},
		{
			name:        "nothing_resolves",
			known:       map[string]*workspace.Workspace{},
			wantWorkers: 0, wantStarted: nil, wantFinished: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeBatch(record("b1", "ws-1", "ws-2"), lookupIn(tt.known))

			assert.Eq(t, got.ID, "b1")
			assert.Eq(t, got.Workers, tt.wantWorkers)
			assert.Eq(t, got.StartedAt, tt.wantStarted)
			assert.Eq(t, got.FinishedAt, tt.wantFinished)
		})
	}
}

func TestBatchChanges(t *testing.T) {
	known := map[string]*workspace.Workspace{
		"ws-1": batchWorkspace("ws-1", &t0, &t1),
		"ws-2": batchWorkspace("ws-2", &t0, &t1),
	}
	diff := func(id string) (*workspace.DiffInfo, error) {
		if id == "ws-2" {
			return nil, errors.New("no git repository")
		}
		return &workspace.DiffInfo{Diff: "+package " + id}, nil
	}

	got := batchChanges(record("b1", "ws-1", "ws-2", "ws-gone"), lookupIn(known), diff)

	assert.Eq(t, len(got), 1)
	assert.Eq(t, got[0].Workspace.ID, "ws-1")
	assert.Eq(t, got[0].Diff.Patch, "+package ws-1")
}

func TestEarliestAndLatest(t *testing.T) {
	assert.Eq(t, earliest(nil, &t1), &t1)
	assert.Eq(t, earliest(&t1, &t0), &t0)
	assert.Eq(t, earliest(&t0, nil), &t0)
	assert.Eq(t, latest(nil, &t1), &t1)
	assert.Eq(t, latest(&t1, &t2), &t2)
	assert.Eq(t, latest(&t2, nil), &t2)
}
