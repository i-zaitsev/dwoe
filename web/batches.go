// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"log/slog"
	"time"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/workspace"
	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

// batchModel reads batches from the workspace manager.
type batchModel struct {
	mgr *workspace.Manager
}

func (m *batchModel) List() ([]data.Batch, error) {
	records, err := m.mgr.BatchRecords()
	if err != nil {
		return nil, err
	}

	batches := make([]data.Batch, 0, len(records))
	for _, rec := range records {
		batches = append(batches, summarizeBatch(rec, m.mgr.Get))
	}

	return batches, nil
}

func (m *batchModel) Get(id string) (data.Batch, error) {
	records, err := m.mgr.BatchRecords()
	if err != nil {
		return data.Batch{}, err
	}

	for _, rec := range records {
		if rec.ID != id {
			continue
		}
		b := summarizeBatch(rec, m.mgr.Get)
		b.Changes = batchChanges(rec, m.mgr.Get, m.mgr.Diff)
		return b, nil
	}

	return data.Batch{}, data.ErrRecordNotFound
}

type batchRef struct {
	ID    string
	Color int
}

// batchIndex maps every workspace ID to its batch and the batch's display color.
func batchIndex(records []*batch.Record) map[string]batchRef {
	index := make(map[string]batchRef)
	for i, rec := range records {
		color := (i % 4) + 1
		for _, entry := range rec.Entries {
			index[entry.WorkspaceID] = batchRef{ID: rec.ID, Color: color}
		}
	}
	return index
}

type lookupFunc func(id string) (*workspace.Workspace, error)

// summarizeBatch collects the summary of a batch record.
// The start time is the earliest start of its workspaces, and the finish time is the latest finish.
// It stays empty until every workspace has finished.
// Only workspaces that still resolve are counted as workers.
func summarizeBatch(rec *batch.Record, lookup lookupFunc) data.Batch {
	b := data.Batch{ID: rec.ID}
	allFinished := true

	for _, entry := range rec.Entries {
		ws, err := lookup(entry.WorkspaceID)
		if err != nil {
			slog.Debug("web: batch workspace", "id", entry.WorkspaceID, "err", err)
			continue
		}
		b.Workers++
		b.StartedAt = earliest(b.StartedAt, ws.StartedAt)
		b.FinishedAt = latest(b.FinishedAt, ws.FinishedAt)
		if ws.FinishedAt == nil {
			allFinished = false
		}
	}

	if !allFinished {
		b.FinishedAt = nil
	}

	return b
}

// batchChanges returns the diffs of the batch workspaces that resolve and have one.
func batchChanges(
	rec *batch.Record,
	lookup lookupFunc,
	diff func(id string) (*workspace.DiffInfo, error),
) []data.BatchChange {
	var changes []data.BatchChange
	for _, entry := range rec.Entries {
		ws, err := lookup(entry.WorkspaceID)
		if err != nil {
			slog.Debug("web: batch workspace", "id", entry.WorkspaceID, "err", err)
			continue
		}
		d, err := diff(entry.WorkspaceID)
		if err != nil {
			slog.Debug("web: batch diff", "id", entry.WorkspaceID, "err", err)
			continue
		}
		changes = append(changes, data.BatchChange{Workspace: summarize(ws), Diff: toDiff(d)})
	}
	return changes
}

func earliest(a, b *time.Time) *time.Time {
	if a == nil || (b != nil && b.Before(*a)) {
		return b
	}
	return a
}

func latest(a, b *time.Time) *time.Time {
	if a == nil || (b != nil && b.After(*a)) {
		return b
	}
	return a
}
