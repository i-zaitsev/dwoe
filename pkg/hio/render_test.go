// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/i-zaitsev/pkg/assert"
)

func minimalUI(content string) fstest.MapFS {
	return fstest.MapFS{
		"templates/layout/base.html":    {Data: []byte(`{{define "base"}}<main>{{template "content" .}}</main>{{end}}`)},
		"templates/pages/home.html":     {Data: []byte(`{{define "content"}}` + content + `{{end}}`)},
		"templates/partials/hello.html": {Data: []byte(`{{define "hello"}}hello {{.}}{{end}}`)},
	}
}

func TestTemplates_PageAndPartial(t *testing.T) {
	tmpl, err := newTemplates(minimalUI("page"), false)
	assert.ErrVal(t, err).Nil().Fatal()

	page, err := tmpl.execute("home.html", "base", nil)
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, page.String(), "<main>page</main>")

	partial, err := tmpl.execute(partialsSet, "hello", "<world>")
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, partial.String(), "hello &lt;world&gt;")
}

func TestTemplates_UnknownSetIsAnError(t *testing.T) {
	tmpl, err := newTemplates(minimalUI("page"), false)
	assert.ErrVal(t, err).Nil().Fatal()

	_, err = tmpl.execute("nope.html", "base", nil)

	assert.ErrVal(t, err).NotNil().Fatal()
}

func TestNewTemplates_InvalidTemplate(t *testing.T) {
	fsys := minimalUI("{{.Broken")

	_, err := newTemplates(fsys, false)

	assert.ErrVal(t, err).Contains("home.html").Fatal()
}

func TestTemplates_ReloadPicksUpChanges(t *testing.T) {
	fsys := minimalUI("before")
	tmpl, err := newTemplates(fsys, true)
	assert.ErrVal(t, err).Nil().Fatal()

	fsys["templates/pages/home.html"] = &fstest.MapFile{Data: []byte(`{{define "content"}}after{{end}}`)}
	page, err := tmpl.execute("home.html", "base", nil)

	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, page.String(), "<main>after</main>")
}

func TestTemplates_CachedWithoutReload(t *testing.T) {
	fsys := minimalUI("before")
	tmpl, err := newTemplates(fsys, false)
	assert.ErrVal(t, err).Nil().Fatal()

	fsys["templates/pages/home.html"] = &fstest.MapFile{Data: []byte(`{{define "content"}}after{{end}}`)}
	page, err := tmpl.execute("home.html", "base", nil)

	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, page.String(), "<main>before</main>")
}

func TestApp_UiDirOverridesEmbeddedFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		path := filepath.Join(dir, name)
		assert.ErrVal(t, os.MkdirAll(filepath.Dir(path), 0o750)).Nil().Fatal()
		assert.ErrVal(t, os.WriteFile(path, []byte(content), 0o600)).Nil().Fatal()
	}
	write("templates/layout/base.html", `{{define "base"}}custom {{template "content" .}}{{end}}`)
	write("templates/pages/error.html", `{{define "content"}}{{.Status}} page{{end}}`)
	write("templates/partials/none.html", `{{define "none"}}{{end}}`)
	write("static/app.css", "body{}")

	app := newTestApp(t, "-ui-dir", dir)

	rr := app.do("/no/such/page")
	assert.Eq(t, rr.Code, http.StatusNotFound)
	assert.Eq(t, rr.Body.String(), "custom 404 page")
	assert.Eq(t, app.do("/static/app.css").Body.String(), "body{}")

	write("templates/pages/error.html", `{{define "content"}}edited{{end}}`)
	assert.Eq(t, app.do("/no/such/page").Body.String(), "custom edited")
}

func TestRender_EmbeddedErrorPage(t *testing.T) {
	app := newTestApp(t)

	rr := app.do("/no/such/page")

	assert.Eq(t, rr.Code, http.StatusNotFound)
	assert.Eq(t, rr.Header().Get("Content-Type"), "text/html; charset=utf-8")
	assert.ContainsAll(t, rr.Body.String(), "<html", "404", `hx-boost="true"`, "/static/htmx.min.js")
}

func TestRender_ThemeCookieSelectsTheme(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/no/such/page", nil)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "light"})
	rr := httptest.NewRecorder()

	app.Routes().ServeHTTP(rr, req)

	assert.Contains(t, rr.Body.String(), `data-theme="light"`)
}

func TestRender_FailureIsLoggedAndNotLeaked(t *testing.T) {
	app := newTestApp(t)
	app.templates = mustTemplates(t, minimalUI("page"))

	rr := app.do("/no/such/page")

	assert.Eq(t, rr.Code, http.StatusInternalServerError)
	assert.ContainsNone(t, rr.Body.String(), "error.html")
	assert.Contains(t, app.logs.String(), "error.html")
}

func TestStaticFiles(t *testing.T) {
	app := newTestApp(t)

	for _, file := range []string{"app.css", "app.js", "htmx.min.js", "htmx-ext-sse.min.js"} {
		t.Run(file, func(t *testing.T) {
			rr := app.do("/static/" + file)

			assert.Eq(t, rr.Code, http.StatusOK)
			assert.True(t, rr.Body.Len() > 0)
		})
	}
}

func TestEmbeddedTemplatesParse(t *testing.T) {
	app := newTestApp(t)

	for _, page := range []string{"workspaces.html", "workspace.html", "batches.html", "batch.html", "error.html", partialsSet} {
		t.Run(page, func(t *testing.T) {
			_, err := app.templates.lookup(page)
			assert.ErrVal(t, err).Nil().Fatal()
		})
	}
}

func mustTemplates(t *testing.T, fsys fstest.MapFS) *templates {
	t.Helper()
	tmpl, err := newTemplates(fsys, false)
	assert.ErrVal(t, err).Nil().Fatal()
	return tmpl
}
