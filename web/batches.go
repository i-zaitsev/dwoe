// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"html/template"
	"net/http"
	"time"

	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// batchSummary stores information collected from workers.
type batchSummary struct {
	ID       string     // batch id
	Workers  int        // number of workers that processed this batch
	Started  *time.Time // earliest time among all worker entries
	Finished *time.Time // latest time among all worker entries
}

// ShortID returns a shortened version of the batch ID.
func (b *batchSummary) ShortID() string {
	if len(b.ID) > 8 {
		return b.ID[:8]
	}
	return b.ID
}

// StartedFmt returns a formatted version of the batch start time.
func (b *batchSummary) StartedFmt() string {
	return fmtTime(b.Started)
}

// FinishedFmt returns a formatted version of the batch finish time.
func (b *batchSummary) FinishedFmt() string {
	return fmtTime(b.Finished)
}

func (b *batchSummary) UpdateStarted(t *time.Time) {
	if t == nil {
		return
	}
	if b.Started == nil || b.Started.After(*t) {
		b.Started = t
	}
}

func (b *batchSummary) UpdateFinished(t *time.Time) {
	if t == nil {
		return
	}
	if b.Finished == nil || b.Finished.Before(*t) {
		b.Finished = t
	}
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

// batchesPageData passed to the rendered template with batch summaries and global page config.
type batchesPageData struct {
	pageConfig
	Batches []batchSummary
}

// batchDiffEntry stores information about a single workspace diff.
type batchDiffEntry struct {
	Name    string
	ID      string
	Commits []workspace.CommitInfo
	Stat    string
	Diff    template.HTML
}

// batchChangesData passed to the rendered template with batch diffs and global page config.
type batchChangesData struct {
	pageConfig
	BatchID string
	Diffs   []batchDiffEntry
}

func (s *Server) listBatches(w http.ResponseWriter, r *http.Request) {
	if batchID := r.URL.Query().Get("batch"); batchID != "" {
		// If requested a specific batch, returns detailed information about
		// the changes for this batch. Otherwise, the endpoint returns the
		// list of batch summaries.
		s.batchChanges(w, r, batchID)
		return
	}

	records, err := s.workspaces.BatchRecords()
	if err != nil {
		records = nil
	}

	var summaries []batchSummary

	// Collect summary information for the batch.
	// The batch start time is equal to the earliest task start time,
	// and the finish time is the latest task finish time. If not
	// all tasks are finished, the batch finish time is left empty.
	for _, rec := range records {
		sum := batchSummary{ID: rec.ID, Workers: len(rec.Entries)}
		allFinished := true
		for _, entry := range rec.Entries {
			ws, errGet := s.workspaces.Get(entry.WorkspaceID)
			if errGet != nil {
				continue
			}
			sum.UpdateStarted(ws.StartedAt)
			sum.UpdateFinished(ws.FinishedAt)
			if ws.FinishedAt == nil {
				allFinished = false
			}
		}
		if !allFinished {
			sum.Finished = nil // filled when all tasks are completed
		}
		summaries = append(summaries, sum)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeTemplate(w, "batches-page", batchesPageData{
		pageConfig: pageConfigFromRequest(r),
		Batches:    summaries,
	})
}

func (s *Server) batchChanges(w http.ResponseWriter, r *http.Request, batchID string) {
	records, err := s.workspaces.BatchRecords()
	if err != nil {
		records = nil
	}

	var diffs []batchDiffEntry
	for _, rec := range records {
		if rec.ID != batchID {
			continue
		}
		for _, entry := range rec.Entries {
			ws, errGet := s.workspaces.Get(entry.WorkspaceID)
			if errGet != nil {
				continue
			}
			diff, errDiff := s.workspaces.Diff(entry.WorkspaceID)
			if errDiff != nil {
				continue
			}
			diffs = append(diffs, batchDiffEntry{
				Name:    ws.Name,
				ID:      ws.ID,
				Commits: diff.Commits,
				Stat:    diff.Stat,
				Diff:    formatDiff(diff.Diff),
			})
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeTemplate(w, "batches-changes", batchChangesData{
		pageConfig: pageConfigFromRequest(r),
		BatchID:    batchID,
		Diffs:      diffs,
	})
}
