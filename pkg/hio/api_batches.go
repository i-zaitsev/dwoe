// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"net/http"
	"time"
)

type batchView struct {
	ID          string     `json:"id"`
	Workers     int        `json:"workers"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	StartedFmt  string     `json:"started_fmt"`
	FinishedFmt string     `json:"finished_fmt"`
}

type batchDiffView struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	diffJSON
}

func (app *App) listBatchesHandler(w http.ResponseWriter, r *http.Request) {
	batches, err := app.models.Batches.List()
	if err != nil {
		app.modelErrorResponse(w, r, err)
		return
	}

	views := make([]batchView, 0, len(batches))
	for _, b := range batches {
		views = append(views, batchView{
			ID:          b.ID,
			Workers:     b.Workers,
			StartedAt:   b.StartedAt,
			FinishedAt:  b.FinishedAt,
			StartedFmt:  fmtTime(b.StartedAt),
			FinishedFmt: fmtTime(b.FinishedAt),
		})
	}

	if err = app.writeJSON(w, http.StatusOK, envelope{"batches": views}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) showBatchHandler(w http.ResponseWriter, r *http.Request) {
	batch, err := app.models.Batches.Get(readRef(r))
	if err != nil {
		app.modelErrorResponse(w, r, err)
		return
	}

	diffs := make([]batchDiffView, 0, len(batch.Changes))
	for _, c := range batch.Changes {
		diffs = append(diffs, batchDiffView{
			Name:     c.Workspace.Name,
			ID:       c.Workspace.ID,
			diffJSON: newDiffJSON(c.Diff),
		})
	}

	if err = app.writeJSON(w, http.StatusOK, envelope{"id": batch.ID, "diffs": diffs}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
