// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"net/http"
	"time"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

type workspaceView struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Exit        string     `json:"exit"`
	Model       string     `json:"model,omitempty"`
	BasePath    string     `json:"base_path"`
	WorkDir     string     `json:"work_dir"`
	ParentID    string     `json:"parent_id,omitempty"`
	BatchID     string     `json:"batch_id,omitempty"`
	BatchColor  int        `json:"batch_color,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	CreatedFmt  string     `json:"created_fmt"`
	StartedFmt  string     `json:"started_fmt"`
	FinishedFmt string     `json:"finished_fmt"`
	Duration    string     `json:"duration"`
	TaskConfig  string     `json:"task_config,omitempty"`
	Prompt      string     `json:"prompt,omitempty"`
}

type commitView struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

type diffJSON struct {
	Commits []commitView `json:"commits"`
	Stat    string       `json:"stat"`
	Diff    string       `json:"diff"`
}

func newWorkspaceView(ws data.Workspace) workspaceView {
	return workspaceView{
		ID:          ws.ID,
		Name:        ws.Name,
		Description: ws.Description,
		Status:      ws.Status,
		Exit:        ws.Exit,
		Model:       ws.Model,
		BasePath:    ws.BasePath,
		WorkDir:     ws.WorkDir,
		ParentID:    ws.ParentID,
		BatchID:     ws.BatchID,
		BatchColor:  ws.BatchColor,
		CreatedAt:   ws.CreatedAt,
		StartedAt:   ws.StartedAt,
		FinishedAt:  ws.FinishedAt,
		CreatedFmt:  fmtTime(ws.CreatedAt),
		StartedFmt:  fmtTime(ws.StartedAt),
		FinishedFmt: fmtTime(ws.FinishedAt),
		Duration:    duration(ws),
		TaskConfig:  ws.TaskConfig,
		Prompt:      ws.Prompt,
	}
}

func newDiffJSON(diff data.Diff) diffJSON {
	commits := make([]commitView, 0, len(diff.Commits))
	for _, c := range diff.Commits {
		commits = append(commits, commitView{Hash: c.Hash, Message: c.Message})
	}
	return diffJSON{Commits: commits, Stat: diff.Stat, Diff: diff.Patch}
}

func (app *App) listWorkspacesHandler(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	filter := data.WorkspaceFilter{Name: readString(qs, "q", ""), Batch: readString(qs, "batch", "")}

	list, err := app.models.Workspaces.List(filter)
	if err != nil {
		app.modelErrorResponse(w, r, err)
		return
	}

	views := make([]workspaceView, 0, len(list))
	for _, ws := range list {
		ws.TaskConfig, ws.Prompt = "", ""
		views = append(views, newWorkspaceView(ws))
	}

	if err = app.writeJSON(w, http.StatusOK, envelope{"workspaces": views}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) showWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := app.models.Workspaces.Get(r.Context(), readRef(r))
	if err != nil {
		app.modelErrorResponse(w, r, err)
		return
	}

	view := newWorkspaceView(ws)
	if err = app.writeJSON(w, http.StatusOK, envelope{"workspace": view}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) diffWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	diff, err := app.models.Workspaces.Diff(readRef(r))
	if err != nil {
		app.modelErrorResponse(w, r, err)
		return
	}

	if err = app.writeJSON(w, http.StatusOK, envelope{"diff": newDiffJSON(diff)}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
