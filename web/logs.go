// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const (
	kb = 1024
	mb = kb * kb
)

type logsPageData struct {
	pageConfig
	Info workspaceInfo
}

func (s *Server) logsWorkspace(w http.ResponseWriter, r *http.Request) {
	info, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}

	rc, err := s.workspaces.Logs(r.Context(), info.ID, false)
	if err != nil {
		http.Error(w, "logs not available", http.StatusNotFound)
		return
	}
	defer rc.Close()

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html")

	matcher := newLineMatcher(r.URL.Query().Get("filter"))
	render := lineRenderer(r)
	scan(rc, func(text string) {
		if !matcher(text) {
			return
		}
		_, _ = fmt.Fprintf(w, "%s\n", render(text))
	})
}

func newLineMatcher(pattern string) func(string) bool {
	if pattern == "" {
		return func(string) bool { return true }
	}
	if re, err := regexp.Compile("(?i)" + pattern); err == nil {
		return re.MatchString
	}
	lower := strings.ToLower(pattern)
	return func(s string) bool { return strings.Contains(strings.ToLower(s), lower) }
}

func (s *Server) logsView(w http.ResponseWriter, r *http.Request) {
	info, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}

	if info.Status == "running" {
		_ = s.workspaces.Sync(r.Context(), info.ID)
		info, _ = s.resolveWorkspace(w, r)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	name := "logs-page"
	if isPretty(r) {
		name = "logs-page-pretty"
	}
	writeTemplate(w, name, logsPageData{
		pageConfig: pageConfigFromRequest(r),
		Info:       info,
	})
}

func (s *Server) logsConnect(w http.ResponseWriter, r *http.Request) {
	if info, ok := s.resolveWorkspace(w, r); ok {
		w.Header().Set("Content-Type", "text/html")
		writeTemplate(w, "stream-connect", info.ID)
	}
}

func (s *Server) logsStream(w http.ResponseWriter, r *http.Request) {
	info, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	_ = s.workspaces.Sync(r.Context(), info.ID)
	info, _ = s.resolveWorkspace(w, r)

	follow := info.Status == "running"
	rc, err := s.workspaces.Logs(r.Context(), info.ID, follow)
	if err != nil {
		writeSSE(w, "done", "completed")
		flusher.Flush()
		return
	}
	defer rc.Close()

	render := lineRenderer(r)
	scan(rc, func(text string) {
		writeSSE(w, "log", render(text))
		flusher.Flush()
	})
	writeSSE(w, "done", "completed")
	flusher.Flush()
}

func isPretty(r *http.Request) bool {
	return r.URL.Query().Get("pretty") == "1"
}

func lineRenderer(r *http.Request) func(string) string {
	if isPretty(r) {
		return renderPrettyLogLine
	}
	return renderLogLine
}

func (s *Server) resolveWorkspace(w http.ResponseWriter, r *http.Request) (workspaceInfo, bool) {
	q := r.URL.Query().Get("q")

	if q == "" {
		http.Error(w, "workspace id or name is required", http.StatusBadRequest)
		return workspaceInfo{}, false
	}

	info, err := s.buildWorkspaceInfo(q)
	if err != nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return workspaceInfo{}, false
	}

	return info, true
}

func writeSSE(w io.Writer, event, data string) {
	if event != "" {
		_, _ = fmt.Fprintf(w, "event: %s\n", event)
	}
	for _, line := range strings.Split(data, "\n") {
		_, _ = fmt.Fprintf(w, "data: %s\n", line)
	}
	_, _ = fmt.Fprint(w, "\n")
}

func renderLogLine(text string) string {
	var buf strings.Builder

	if !json.Valid([]byte(text)) {
		writeTemplate(&buf, "log-line", text)
		return buf.String()
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(text), "", "  "); err != nil {
		return "[failed]"
	}

	writeTemplate(&buf, "log-line-json", struct {
		Raw    string
		Pretty template.HTML
	}{
		Raw:    text,
		Pretty: template.HTML(highlightJSON(pretty.String())),
	})

	return buf.String()
}

func scan(rc io.ReadCloser, callback func(text string)) {
	sc := bufio.NewScanner(rc)
	sc.Buffer(make([]byte, 0, 64*kb), mb)
	for sc.Scan() {
		callback(sc.Text())
	}
}
