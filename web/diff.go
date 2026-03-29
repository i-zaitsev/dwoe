// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"html"
	"html/template"
	"net/http"
	"strings"

	"github.com/i-zaitsev/dwoe/internal/workspace"
)

type diffView struct {
	Commits []workspace.CommitInfo
	Stat    string
	Diff    template.HTML
}

func (s *Server) diffWorkspace(w http.ResponseWriter, r *http.Request) {
	info, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}

	diff, err := s.workspaces.Diff(info.ID)
	if err != nil {
		http.Error(w, "diff not available", http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html")

	writeTemplate(w, "diff-info", diffView{
		Commits: diff.Commits,
		Stat:    diff.Stat,
		Diff:    formatDiff(diff.Diff),
	})
}

func formatDiff(raw string) template.HTML {
	var buf strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		escaped := html.EscapeString(line)
		switch {
		case strings.HasPrefix(line, "+"):
			buf.WriteString(`<span class="da">`)
			buf.WriteString(escaped)
			buf.WriteString("</span>\n")
		case strings.HasPrefix(line, "-"):
			buf.WriteString(`<span class="dd">`)
			buf.WriteString(escaped)
			buf.WriteString("</span>\n")
		case strings.HasPrefix(line, "@@"):
			buf.WriteString(`<span class="dh">`)
			buf.WriteString(escaped)
			buf.WriteString("</span>\n")
		default:
			buf.WriteString(escaped)
			buf.WriteByte('\n')
		}
	}
	return template.HTML(buf.String())
}
