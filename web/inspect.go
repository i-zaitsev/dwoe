// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/i-zaitsev/dwoe/internal/workspace"
	"gopkg.in/yaml.v3"
)

type workspaceInfo struct {
	ID            string
	Name          string
	Status        string
	Exit          string
	Model         string
	BasePath      string
	WorkDir       string
	BatchID       string
	BatchColor    int
	CreatedAt     *time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
	TaskConfig    string
	PromptContent string
}

func (w workspaceInfo) BatchDisplay() string {
	if len(w.BatchID) > 8 {
		return w.BatchID[:8]
	}
	return w.BatchID
}

func (w workspaceInfo) StartedFmt() string {
	if w.StartedAt == nil {
		return "-"
	}
	return w.StartedAt.Format("15:04 Jan 02 2006")
}

func (w workspaceInfo) Duration() string {
	if w.StartedAt == nil {
		return "not started"
	}
	if w.FinishedAt == nil {
		return "running"
	}
	d := w.FinishedAt.Sub(*w.StartedAt)
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm %ds", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
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
	info := toWorkspaceInfo(ws)
	if ws.Config != nil {
		info.PromptContent = resolvePromptContent(ws)
	}
	return info, nil
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
		if data, err := yaml.Marshal(ws.Config); err == nil {
			info.TaskConfig = string(data)
		} else {
			info.TaskConfig = "failed to marshal task config"
		}
	}
	return info
}

// resolvePromptContent returns the task prompt text.
// It prefers the inline TaskPrompt field; when empty, reads the prompt file.
// Absolute paths and path traversal are rejected to keep reads scoped to the workspace.
func resolvePromptContent(ws *workspace.Workspace) string {

	strPrompt := ws.Config.Agent.TaskPrompt
	if strPrompt != "" {
		// prompt was provided as an inline string argument to the task running command
		return strPrompt
	}

	// otherwise, the prompt comes from a workspace file
	filePath := ws.Config.Source.PromptFile
	if filePath == "" {
		return ""
	}

	// Reject absolute paths to keep reads scoped to the workspace.
	// This condition should not happen for a valid task config,
	// but doing an extra check to avoid any potential failures
	if filepath.IsAbs(filePath) {
		slog.Warn("absolute prompt file path rejected", "path", filePath, "workspaceID", ws.ID)
		return ""
	}

	// Reject a relative path that escapes the workspace directory.
	// Another guard against a potentially misconfigured task path
	// that cli should capture.
	fullPath, ok := inWorkDir(ws.WorkDir(), filePath)
	if !ok {
		slog.Warn("prompt file path with parent dir rejected", "path", filePath, "workspaceID", ws.ID)
		return ""
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}

	return string(data)
}

func inWorkDir(workDir, filePath string) (string, bool) {
	full := filepath.Join(workDir, filePath)
	rel, err := filepath.Rel(workDir, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return full, true
}
