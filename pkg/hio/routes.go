// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import "net/http"

// Routes returns the handler of the whole dashboard: pages, static files, and the API.
func (app *App) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(app.static)))
	mux.HandleFunc("GET /theme", app.themeHandler)

	mux.HandleFunc("GET /{$}", app.listWorkspacesPage)
	mux.HandleFunc("GET /workspaces/{id}", app.workspacePage)
	mux.HandleFunc("GET /workspaces/{id}/info", app.workspaceInfoPartial)
	mux.HandleFunc("GET /workspaces/{id}/diff", app.diffPartial)
	mux.HandleFunc("GET /workspaces/{id}/logs", app.logsPartial)
	mux.HandleFunc("GET /workspaces/{id}/logs/stream", app.logsStreamPage)
	mux.HandleFunc("GET /batches", app.batchesPage)
	mux.HandleFunc("GET /batches/{id}", app.batchPage)

	mux.Handle("/api/", app.apiRoutes())
	mux.HandleFunc("/", app.pageNotFound)

	return app.recoverPanic(app.logRequest(secureHeaders(mux)))
}

func (app *App) apiRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/healthcheck", app.healthcheckHandler)
	mux.HandleFunc("GET /api/v1/workspaces", app.listWorkspacesHandler)
	mux.HandleFunc("GET /api/v1/workspaces/{id}", app.showWorkspaceHandler)
	mux.HandleFunc("GET /api/v1/workspaces/{id}/diff", app.diffWorkspaceHandler)
	mux.HandleFunc("GET /api/v1/workspaces/{id}/logs", app.logsHandler)
	mux.HandleFunc("GET /api/v1/workspaces/{id}/logs/stream", app.logsStreamHandler)
	mux.HandleFunc("GET /api/v1/batches", app.listBatchesHandler)
	mux.HandleFunc("GET /api/v1/batches/{id}", app.showBatchHandler)
	mux.HandleFunc("/", app.notFoundResponse)

	return cacheControl(mux)
}
