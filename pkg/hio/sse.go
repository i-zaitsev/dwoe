// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// startSSE prepares a response for server-sent events and lifts the write deadline.
func (app *App) startSSE(w http.ResponseWriter) *http.ResponseController {
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		app.Log().Debug("clear write deadline", "err", err)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	return rc
}

func writeSSE(w io.Writer, event, data string) {
	if event != "" {
		_, _ = fmt.Fprintf(w, "event: %s\n", event)
	}
	for _, line := range strings.Split(data, "\n") {
		_, _ = fmt.Fprintf(w, "data: %s\n", line)
	}
	_, _ = fmt.Fprint(w, "\n")
}

const logStreamErrorMessage = "the log stream failed"

// logStreamFailed logs a failed log stream and reports whether the client
// should get an error event. A cancelled request is not an error.
func (app *App) logStreamFailed(r *http.Request, id string, err error) bool {
	if r.Context().Err() != nil {
		return false
	}
	app.Log().Error("workspace logs", "id", id, "err", err)
	return true
}
