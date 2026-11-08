// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import "net/http"

type errorPage struct {
	layoutData
	Status     int
	StatusText string
	Message    string
}

func (app *App) pageError(w http.ResponseWriter, r *http.Request, status int, message string) {
	err := app.writePage(w, status, "error.html", errorPage{
		layoutData: app.layout(r, "Error", ""),
		Status:     status,
		StatusText: http.StatusText(status),
		Message:    message,
	})
	if err != nil {
		app.plainError(w, r, err)
	}
}

func (app *App) partialError(w http.ResponseWriter, r *http.Request, status int, message string) {
	if err := app.writePartial(w, status, "error", message); err != nil {
		app.plainError(w, r, err)
	}
}

func (app *App) plainError(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	http.Error(w, serverErrorMessage, http.StatusInternalServerError)
}

func (app *App) pageNotFound(w http.ResponseWriter, r *http.Request) {
	app.pageError(w, r, http.StatusNotFound, "the requested page is not found")
}

func (app *App) serverError(w http.ResponseWriter, r *http.Request, err error) {
	if isAPI(r) {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.logError(r, err)
	if isHTMX(r) {
		app.partialError(w, r, http.StatusInternalServerError, serverErrorMessage)
		return
	}
	app.pageError(w, r, http.StatusInternalServerError, serverErrorMessage)
}

func (app *App) modelError(w http.ResponseWriter, r *http.Request, err error) {
	if isMissing(err) {
		app.pageNotFound(w, r)
		return
	}
	app.serverError(w, r, err)
}

func (app *App) partialModelError(w http.ResponseWriter, r *http.Request, err error) {
	if isMissing(err) {
		app.partialError(w, r, http.StatusNotFound, "Not available.")
		return
	}
	app.logError(r, err)
	app.partialError(w, r, http.StatusInternalServerError, serverErrorMessage)
}
