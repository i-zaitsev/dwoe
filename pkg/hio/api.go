// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import "net/http"

func (app *App) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	if err := app.writeJSON(w, http.StatusOK, envelope{"status": "available"}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
