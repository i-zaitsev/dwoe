// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package hio serves a workspace dashboard: HTML pages enhanced with htmx and
// a JSON API, both reading through the models of package data.
package hio

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/i-zaitsev/dwoe/pkg/hio/config"
	"github.com/i-zaitsev/dwoe/pkg/hio/data"
	"github.com/i-zaitsev/dwoe/pkg/hio/ui"
)

// App holds the dependencies of the handlers.
type App struct {
	env       *config.Env
	models    data.Models
	templates *templates
	static    fs.FS
}

// Log returns the logger of the application.
func (app *App) Log() *slog.Logger {
	return app.env.AppLog()
}

// NewApp creates an App. The templates and static files are embedded unless
// the environment names a UI directory, which is then re-read on every request.
func NewApp(env *config.Env, models data.Models) (*App, error) {
	fileSys := fs.FS(ui.FS)

	reload := false
	if dir := env.UIDir(); dir != "" {
		fileSys = os.DirFS(dir)
		reload = true
	}

	tmpl, err := newTemplates(fileSys, reload)
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}

	static, err := fs.Sub(fileSys, "static")
	if err != nil {
		return nil, fmt.Errorf("load static files: %w", err)
	}

	return &App{env: env, models: models, templates: tmpl, static: static}, nil
}
