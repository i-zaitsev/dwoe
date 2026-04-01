// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package batch

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestBatch_BranchName(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		want, taskFile string
	}{
		{"agent/boilerplate", "tasks/agent/boilerplate.yaml"},
		{"refactor/auth-system", "backlog/refactor/auth-system.yaml"},
		{"ad-hoc-job", "ad-hoc-job.yaml"},
	}
	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			got := BranchName(tc.taskFile)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("BranchName() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBatch_SaveRecord(t *testing.T) {
	t.Parallel()
	rec := &Record{
		ID:        "batch-1",
		SourceDir: "/path/to/source",
		CreatedAt: "2026-03-03T12:00:00Z",
		Entries: []Entry{
			{TaskFile: "task-one.yaml", WorkspaceID: "ws-1", Branch: "agent/one"},
			{TaskFile: "task-two.yaml", WorkspaceID: "ws-2", Branch: "agent/two"},
		},
	}
	dataDir := t.TempDir()
	assert.NotErr(t, SaveRecord(dataDir, rec))

	got, err := LoadRecord(dataDir, "batch-1")
	assert.NotErr(t, err)

	if diff := cmp.Diff(rec, got); diff != "" {
		t.Errorf("round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestBatch_LoadRecord(t *testing.T) {
	t.Parallel()
	got, err := LoadRecord("testdata", "test-batch")
	assert.NotErr(t, err)

	want := &Record{
		ID:        "test-batch",
		SourceDir: "/tmp/source",
		CreatedAt: "2026-01-15T10:30:00Z",
		Entries: []Entry{
			{TaskFile: "tasks/agent/one.yaml", WorkspaceID: "ws-1", Branch: "agent/one"},
			{TaskFile: "tasks/agent/two.yaml", WorkspaceID: "ws-2", Branch: "agent/two"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LoadRecord() mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadOrCreate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	id := "test-batch-id"

	t.Run("creates_when_missing", func(t *testing.T) {
		rec, err := LoadOrCreate(dir, id, "/some/repo")
		assert.NotErr(t, err)
		assert.Equal(t, id, rec.ID)
		assert.Equal(t, rec.SourceDir, "/some/repo")
		assert.Zero(t, len(rec.Entries))
	})

	t.Run("loads_existing", func(t *testing.T) {
		rec, _ := LoadOrCreate(dir, id, "/some/repo")
		rec.Entries = append(rec.Entries, Entry{WorkspaceID: "ws-1"})
		assert.NotErr(t, SaveRecord(dir, rec))

		loaded, err := LoadOrCreate(dir, id, "/some/repo")
		assert.NotErr(t, err)
		assert.Equal(t, len(loaded.Entries), 1)
	})
}

func TestBatch_LoadRecord_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		dataDir string
		id      string
	}{
		{"not_found", "testdata", "nonexistent"},
		{"corrupt_json", "testdata", "corrupt"},
		{"empty_datadir", t.TempDir(), "anything"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := LoadRecord(tt.dataDir, tt.id)
			assert.Err(t, err)
		})
	}
}

func TestBatch_ReadRecords(t *testing.T) {
	t.Parallel()
	recs, err := ReadRecords("testdata", "batches_many")
	assert.NotErr(t, err)
	assert.Equal(t, 3, len(recs))
}
