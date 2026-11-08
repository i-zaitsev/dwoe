// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"fmt"
	"net/http"
)

func (app *App) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	if err := app.writeJSON(w, status, envelope{"error": message}, nil); err != nil {
		app.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *App) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	app.errorResponse(w, r, http.StatusInternalServerError, serverErrorMessage)
}

func (app *App) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusNotFound, "the requested resource is not found")
}

func (app *App) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("the %s method is not supported", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, msg)
}

func (app *App) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (app *App) modelErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	if isMissing(err) {
		app.notFoundResponse(w, r)
		return
	}
	app.serverErrorResponse(w, r, err)
}
