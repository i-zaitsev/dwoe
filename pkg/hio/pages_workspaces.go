// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"net/http"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

type workspacesPageData struct {
	layoutData
	Workspaces []data.Workspace
	Query      string
	Batch      string
}

type workspacePageData struct {
	layoutData
	Workspace data.Workspace
}

// listWorkspacesPage lists workspaces. A filter request from htmx gets only the table.
func (app *App) listWorkspacesPage(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	filter := data.WorkspaceFilter{Name: readString(qs, "q", ""), Batch: readString(qs, "batch", "")}

	list, err := app.models.Workspaces.List(filter)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	page := workspacesPageData{
		layoutData: app.layout(r, "Workspaces", "workspaces"),
		Workspaces: list,
		Query:      filter.Name,
		Batch:      filter.Batch,
	}

	w.Header().Add("Vary", "HX-Request, HX-Target")
	if isFragment(r, "results") {
		app.renderPartial(w, r, http.StatusOK, "workspace-table", page)
		return
	}
	app.render(w, r, http.StatusOK, "workspaces.html", page)
}

func (app *App) workspacePage(w http.ResponseWriter, r *http.Request) {
	ws, err := app.models.Workspaces.Get(r.Context(), readRef(r))
	if err != nil {
		app.modelError(w, r, err)
		return
	}

	app.render(w, r, http.StatusOK, "workspace.html", workspacePageData{
		layoutData: app.layout(r, ws.Name, "workspaces"),
		Workspace:  ws,
	})
}

type workspaceInfoError struct {
	ID      string
	Message string
}

// workspaceInfoPartial answers the info poll. A failed poll renders a panel that
// keeps polling, so the next good answer restores it.
func (app *App) workspaceInfoPartial(w http.ResponseWriter, r *http.Request) {
	ws, err := app.models.Workspaces.Get(r.Context(), readRef(r))
	if err != nil {
		if isMissing(err) {
			app.partialModelError(w, r, err)
			return
		}
		app.logError(r, err)
		errPanel := workspaceInfoError{ID: readRef(r), Message: serverErrorMessage}
		if errRender := app.writePartial(w, http.StatusInternalServerError, "workspace-info-error", errPanel); errRender != nil {
			app.plainError(w, r, errRender)
		}
		return
	}

	app.renderPartial(w, r, http.StatusOK, "workspace-info", ws)
}

func (app *App) diffPartial(w http.ResponseWriter, r *http.Request) {
	diff, err := app.models.Workspaces.Diff(readRef(r))
	if err != nil {
		app.partialModelError(w, r, err)
		return
	}

	app.renderPartial(w, r, http.StatusOK, "diff", diff)
}

// logsPartial renders the whole log of a finished workspace, optionally filtered.
func (app *App) logsPartial(w http.ResponseWriter, r *http.Request) {
	match := newEventMatcher(readString(r.URL.Query(), "filter", ""))

	var events []data.LogEvent
	err := app.models.Workspaces.Logs(r.Context(), readRef(r), false, func(ev data.LogEvent) error {
		if match(ev) {
			events = append(events, ev)
		}
		return nil
	})
	if err != nil {
		app.partialModelError(w, r, err)
		return
	}

	app.renderPartial(w, r, http.StatusOK, "log-rows", events)
}

// logsStreamPage follows the log of a running workspace as server-sent HTML rows.
func (app *App) logsStreamPage(w http.ResponseWriter, r *http.Request) {
	ws, err := app.models.Workspaces.Get(r.Context(), readRef(r))
	if err != nil {
		app.partialModelError(w, r, err)
		return
	}

	rc := app.startSSE(w)

	err = app.models.Workspaces.Logs(r.Context(), ws.ID, ws.Running(), func(ev data.LogEvent) error {
		row, errRender := app.renderString("log-row", ev)
		if errRender != nil {
			app.Log().Error("render log row", "err", errRender)
			return nil
		}
		writeSSE(w, "log", row)
		return rc.Flush()
	})
	if err != nil {
		app.Log().Debug("workspace logs", "id", ws.ID, "err", err)
	}

	writeSSE(w, "done", "completed")
	_ = rc.Flush()
}
