// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"embed"
	"html/template"
	"io"
	"log/slog"
)

//go:embed templates/*.html
var templateFS embed.FS

var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

func writeTemplate(w io.Writer, name string, data any) {
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("web: template", "name", name, "err", err)
	}
}
