// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"net/http"
	"net/url"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

type batchesPageData struct {
	layoutData
	Batches []data.Batch
}

type batchPageData struct {
	layoutData
	Batch data.Batch
}

func (app *App) batchesPage(w http.ResponseWriter, r *http.Request) {
	batches, err := app.models.Batches.List()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.render(w, r, http.StatusOK, "batches.html", batchesPageData{
		layoutData: app.layout(r, "Batches", "batches"),
		Batches:    batches,
	})
}

func (app *App) batchPage(w http.ResponseWriter, r *http.Request) {
	batch, err := app.models.Batches.Get(readRef(r))
	if err != nil {
		app.modelError(w, r, err)
		return
	}

	app.render(w, r, http.StatusOK, "batch.html", batchPageData{
		layoutData: app.layout(r, "Batch "+shortID(batch.ID), "batches"),
		Batch:      batch,
	})
}

// themeHandler stores the theme in a cookie and returns to the page it was called from.
func (app *App) themeHandler(w http.ResponseWriter, r *http.Request) {
	theme := "dark"
	if r.URL.Query().Get("v") == "light" {
		theme = "light"
	}
	http.SetCookie(w, &http.Cookie{Name: "theme", Value: theme, Path: "/", SameSite: http.SameSiteLaxMode})

	target := "/"
	if ref, err := url.Parse(r.Referer()); err == nil && ref.Host == r.Host && ref.Path != "" {
		target = ref.RequestURI()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
