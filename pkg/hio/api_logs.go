// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"encoding/json"
	"net/http"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

type logEventView struct {
	Time   string          `json:"time,omitempty"`
	Kind   string          `json:"kind"`
	Text   string          `json:"text"`
	Detail string          `json:"detail,omitempty"`
	Raw    json.RawMessage `json:"raw,omitempty"`
}

func newLogEventView(ev data.LogEvent) logEventView {
	return logEventView{Time: ev.Time, Kind: ev.Kind, Text: ev.Text, Detail: ev.Detail, Raw: ev.Raw}
}

func (app *App) logsHandler(w http.ResponseWriter, r *http.Request) {
	match := newEventMatcher(readString(r.URL.Query(), "filter", ""))

	events := make([]logEventView, 0)
	err := app.models.Workspaces.Logs(r.Context(), readRef(r), false, func(ev data.LogEvent) error {
		if match(ev) {
			events = append(events, newLogEventView(ev))
		}
		return nil
	})
	if err != nil {
		app.modelErrorResponse(w, r, err)
		return
	}

	if err = app.writeJSON(w, http.StatusOK, envelope{"events": events}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) logsStreamHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := app.models.Workspaces.Get(r.Context(), readRef(r))
	if err != nil {
		app.modelErrorResponse(w, r, err)
		return
	}

	rc := app.startSSE(w)

	err = app.models.Workspaces.Logs(r.Context(), ws.ID, ws.Running(), func(ev data.LogEvent) error {
		payload, errMarshal := json.Marshal(newLogEventView(ev))
		if errMarshal != nil {
			app.Log().Debug("marshal log event", "err", errMarshal)
			return nil
		}
		writeSSE(w, "log", string(payload))
		return rc.Flush()
	})
	if err != nil {
		app.Log().Debug("workspace logs", "id", ws.ID, "err", err)
	}

	writeSSE(w, "done", "completed")
	_ = rc.Flush()
}
