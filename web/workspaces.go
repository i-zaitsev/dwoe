// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

const (
	kb = 1024
	mb = kb * kb

	syncTimeout = 10 * time.Second
)

// workspaceModel reads workspaces from the workspace manager.
type workspaceModel struct {
	mgr *workspace.Manager
}

// resolve finds a workspace by ID first and by name next.
func (m *workspaceModel) resolve(ref string) (*workspace.Workspace, error) {
	return lookup(ref, m.mgr.Get, m.mgr.GetByName)
}

// lookup tries getID first and falls back to getName only on a not-found error.
func lookup(ref string, getID, getName func(string) (*workspace.Workspace, error)) (*workspace.Workspace, error) {
	ws, err := getID(ref)
	var missing *state.NotFoundError
	if errors.As(err, &missing) {
		ws, err = getName(ref)
	}
	if err != nil {
		return nil, notFound(err)
	}
	return ws, nil
}

func (m *workspaceModel) index() map[string]batchRef {
	records, err := m.mgr.BatchRecords()
	if err != nil {
		slog.Debug("web: batch records", "err", err)
		return nil
	}
	return batchIndex(records)
}

// Get syncs a running workspace with its container before reading it.
func (m *workspaceModel) Get(ctx context.Context, ref string) (data.Workspace, error) {
	ws, err := m.resolve(ref)
	if err != nil {
		return data.Workspace{}, err
	}

	if ws.Status == workspace.StatusRunning {
		syncCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), syncTimeout)
		defer cancel()
		if err = m.mgr.Sync(syncCtx, ws.ID); err != nil {
			slog.Debug("web: workspace sync", "id", ws.ID, "err", err)
		} else if fresh, errGet := m.mgr.Get(ws.ID); errGet == nil {
			ws = fresh
		}
	}

	w := describe(ws)
	b := m.index()[ws.ID]
	w.BatchID, w.BatchColor = b.ID, b.Color

	return w, nil
}

func (m *workspaceModel) List(f data.WorkspaceFilter) ([]data.Workspace, error) {
	list, err := m.mgr.List()
	if err != nil {
		return nil, err
	}

	batches := m.index()

	var out []data.Workspace
	for _, ws := range list {
		b := batches[ws.ID]
		if !strings.Contains(ws.Name, f.Name) || (f.Batch != "" && b.ID != f.Batch) {
			continue
		}
		w := summarize(ws)
		w.BatchID, w.BatchColor = b.ID, b.Color
		out = append(out, w)
	}

	return out, nil
}

func (m *workspaceModel) Diff(ref string) (data.Diff, error) {
	ws, err := m.resolve(ref)
	if err != nil {
		return data.Diff{}, err
	}

	diff, err := m.mgr.Diff(ws.ID)
	if err != nil {
		return data.Diff{}, fmt.Errorf("%w: %v", data.ErrUnavailable, err)
	}

	return toDiff(diff), nil
}

func (m *workspaceModel) Logs(ctx context.Context, ref string, follow bool, emit func(data.LogEvent) error) error {
	ws, err := m.resolve(ref)
	if err != nil {
		return err
	}

	rc, err := m.mgr.Logs(ctx, ws.ID, follow)
	if err != nil {
		return fmt.Errorf("%w: %v", data.ErrUnavailable, err)
	}
	defer func() { _ = rc.Close() }()

	if err = emitEvents(rc, emit); err != nil && ctx.Err() == nil {
		return err
	}

	return nil
}

// emitEvents parses every log line read from r and passes its events to emit.
// It stops at the first error returned by emit.
func emitEvents(r io.Reader, emit func(data.LogEvent) error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*kb), mb)
	for sc.Scan() {
		for _, ev := range parseLogEvents(sc.Text()) {
			if err := emit(ev); err != nil {
				return err
			}
		}
	}
	return sc.Err()
}

// notFound maps the state store's not-found error to the data layer's.
func notFound(err error) error {
	var missing *state.NotFoundError
	if errors.As(err, &missing) {
		return data.ErrRecordNotFound
	}
	return err
}

// summarize converts a workspace to the fields shown in lists.
func summarize(ws *workspace.Workspace) data.Workspace {
	w := data.Workspace{
		ID:         ws.ID,
		Name:       ws.Name,
		Status:     ws.Status,
		Exit:       ws.ExitStatus(),
		BasePath:   ws.BasePath,
		WorkDir:    ws.WorkDir(),
		ParentID:   ws.ParentID,
		CreatedAt:  ws.CreatedAt,
		StartedAt:  ws.StartedAt,
		FinishedAt: ws.FinishedAt,
	}
	if ws.Config != nil {
		w.Description = ws.Config.Description
		if ws.Config.Agent != nil {
			w.Model = ws.Config.Agent.Provider.Model
		}
	}
	return w
}

// describe converts a workspace to all its fields, including the task config and the prompt.
func describe(ws *workspace.Workspace) data.Workspace {
	w := summarize(ws)
	if ws.Config == nil {
		return w
	}

	if out, err := yaml.Marshal(ws.Config); err == nil {
		w.TaskConfig = string(out)
	} else {
		w.TaskConfig = "failed to marshal task config"
	}
	w.Prompt = resolvePromptContent(ws)

	return w
}

func toDiff(diff *workspace.DiffInfo) data.Diff {
	commits := make([]data.Commit, 0, len(diff.Commits))
	for _, c := range diff.Commits {
		commits = append(commits, data.Commit{Hash: c.Hash, Message: c.Message})
	}
	return data.Diff{Commits: commits, Stat: diff.Stat, Patch: diff.Diff}
}

// resolvePromptContent returns the task prompt text.
// It prefers the inline TaskPrompt field; when empty, reads the prompt file.
// Absolute paths and path traversal are rejected to keep reads scoped to the workspace.
func resolvePromptContent(ws *workspace.Workspace) string {
	if ws.Config.Agent != nil && ws.Config.Agent.TaskPrompt != "" {
		// prompt was provided as an inline string argument to the task running command
		return ws.Config.Agent.TaskPrompt
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

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}

	return string(content)
}

func inWorkDir(workDir, filePath string) (string, bool) {
	full := filepath.Join(workDir, filePath)
	rel, err := filepath.Rel(workDir, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return full, true
}
