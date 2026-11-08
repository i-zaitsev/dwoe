// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
)

const (
	baseTemplate = "templates/layout/base.html"
	partialsSet  = "partials"
)

type layoutData struct {
	Title  string
	Active string
	Theme  string
}

// templates holds one template set per page.
// Every set contains the layout, all partials, and the page itself.
// The definitions never collide. The partials also form a set of their
// own for htmx fragments.
type templates struct {
	fsys   fs.FS
	reload bool
	cache  map[string]*template.Template
}

func newTemplates(fsys fs.FS, reload bool) (*templates, error) {
	cache, err := buildTemplates(fsys)
	if err != nil {
		return nil, err
	}
	return &templates{fsys: fsys, reload: reload, cache: cache}, nil
}

func buildTemplates(fsys fs.FS) (map[string]*template.Template, error) {
	partials, err := fs.Glob(fsys, "templates/partials/*.html")
	if err != nil {
		return nil, err
	}
	pages, err := fs.Glob(fsys, "templates/pages/*.html")
	if err != nil {
		return nil, err
	}

	cache := make(map[string]*template.Template, len(pages)+1)

	for _, page := range pages {
		name := path.Base(page)
		files := append([]string{baseTemplate}, partials...)
		files = append(files, page)
		tmpl, errParse := template.New(name).Funcs(funcMap).ParseFS(fsys, files...)
		if errParse != nil {
			return nil, fmt.Errorf("parse %s: %w", name, errParse)
		}
		cache[name] = tmpl
	}

	set, err := template.New(partialsSet).Funcs(funcMap).ParseFS(fsys, partials...)
	if err != nil {
		return nil, fmt.Errorf("parse partials: %w", err)
	}
	cache[partialsSet] = set

	return cache, nil
}

func (t *templates) lookup(set string) (*template.Template, error) {
	cache := t.cache
	if t.reload {
		var err error
		if cache, err = buildTemplates(t.fsys); err != nil {
			return nil, err
		}
	}

	tmpl, ok := cache[set]
	if !ok {
		return nil, fmt.Errorf("template set %q not found", set)
	}

	return tmpl, nil
}

func (t *templates) execute(set, name string, data any) (*bytes.Buffer, error) {
	tmpl, err := t.lookup(set)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, fmt.Errorf("execute %s: %w", name, err)
	}

	return &buf, nil
}

func (app *App) layout(r *http.Request, title, active string) layoutData {
	theme := "dark"
	if c, err := r.Cookie("theme"); err == nil && c.Value == "light" {
		theme = "light"
	}
	return layoutData{Title: title, Active: active, Theme: theme}
}

// render writes a full page, or reports the failure through serverError.
func (app *App) render(w http.ResponseWriter, r *http.Request, status int, page string, data any) {
	if err := app.writePage(w, status, page, data); err != nil {
		app.serverError(w, r, err)
	}
}

// renderPartial writes an htmx fragment, or reports the failure through serverError.
func (app *App) renderPartial(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	if err := app.writePartial(w, status, name, data); err != nil {
		app.serverError(w, r, err)
	}
}

// writePage and writePartial return the template error instead of handling it,
// so the error helpers can use them without recursing into a failing template.
func (app *App) writePage(w http.ResponseWriter, status int, page string, data any) error {
	buf, err := app.templates.execute(page, "base", data)
	if err != nil {
		return err
	}
	writeHTML(w, status, buf.Bytes())
	return nil
}

func (app *App) writePartial(w http.ResponseWriter, status int, name string, data any) error {
	buf, err := app.templates.execute(partialsSet, name, data)
	if err != nil {
		return err
	}
	writeHTML(w, status, buf.Bytes())
	return nil
}

// renderString renders a fragment to a string, for server-sent events.
func (app *App) renderString(name string, data any) (string, error) {
	buf, err := app.templates.execute(partialsSet, name, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
