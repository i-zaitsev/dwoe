// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/i-zaitsev/dwoe/internal/workspace"
)

type workspaceInfo struct {
	ID         string
	Name       string
	Status     string
	Exit       string
	Model      string
	BasePath   string
	WorkDir    string
	BatchID    string
	BatchColor int
	CreatedAt  *time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

func (w workspaceInfo) BatchDisplay() string {
	if len(w.BatchID) > 8 {
		return w.BatchID[:8]
	}
	return w.BatchID
}

func (w workspaceInfo) ShortTime() string {
	if w.StartedAt == nil {
		return "-"
	}
	s := w.StartedAt.Format("2006-01-02 15:04:05")
	if len(s) >= 16 {
		return s[11:16]
	}
	return s
}

func (s *Server) inspectWorkspace(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "workspace id or name is required", http.StatusBadRequest)
		return
	}

	_ = s.workspaces.Sync(r.Context(), q)
	info, err := s.buildWorkspaceInfo(q)

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html")

	if err != nil {
		_, _ = fmt.Fprintf(w, `<div class="error">Workspace not found</div>`)
		return
	}
	writeTemplate(w, "workspace-info", info)
}

func (s *Server) buildWorkspaceInfo(q string) (workspaceInfo, error) {
	ws, err := s.workspaces.Get(q)
	if err != nil {
		ws, err = s.workspaces.GetByName(q)
	}
	if err != nil {
		return workspaceInfo{}, err
	}
	return toWorkspaceInfo(ws), nil
}

func toWorkspaceInfo(ws *workspace.Workspace) workspaceInfo {
	info := workspaceInfo{
		ID:         ws.ID,
		Name:       ws.Name,
		Status:     ws.Status,
		Exit:       ws.ExitStatus(),
		BasePath:   ws.BasePath,
		WorkDir:    ws.WorkDir(),
		CreatedAt:  ws.CreatedAt,
		StartedAt:  ws.StartedAt,
		FinishedAt: ws.FinishedAt,
	}
	if ws.Config != nil {
		info.Model = ws.Config.Agent.Model
	}
	return info
}
