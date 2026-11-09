// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/i-zaitsev/pkg/assert"

	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/schema/schematest"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

var errStop = errors.New("stop")

func fixtureWorkspace(cfg *config.Task) *workspace.Workspace {
	started := time.Date(2025, time.March, 15, 9, 30, 0, 0, time.UTC)
	sw := state.EmptyWorkspace("ws-alpha", "alpha")
	sw.Status = workspace.StatusCompleted
	sw.BasePath = "testdata/ws-alpha"
	sw.StartedAt = &started
	return workspace.New(sw, cfg)
}

func TestSummarize(t *testing.T) {
	ws := fixtureWorkspace(&config.Task{
		Description: "health endpoint",
		Agent:       &config.Agent{TaskPrompt: "inline"},
	})

	got := summarize(ws)

	assert.Eq(t, got.ID, "ws-alpha")
	assert.Eq(t, got.Name, "alpha")
	assert.Eq(t, got.Status, "completed")
	assert.Eq(t, got.Description, "health endpoint")
	assert.Eq(t, got.BasePath, "testdata/ws-alpha")
	assert.Eq(t, got.StartedAt, ws.StartedAt)
	assert.Eq(t, got.TaskConfig, "")
	assert.Eq(t, got.Prompt, "")
}

func TestSummarize_WithoutConfig(t *testing.T) {
	got := summarize(fixtureWorkspace(nil))

	assert.Eq(t, got.Description, "")
	assert.Eq(t, got.Model, "")
}

func TestDescribe_ReadsPromptFileFromFixture(t *testing.T) {
	ws := fixtureWorkspace(&config.Task{Source: config.Source{PromptFile: "prompt.md"}})

	got := describe(ws)

	assert.Eq(t, got.Prompt, "Add a /health endpoint.\n")
	assert.Contains(t, got.TaskConfig, "prompt.md")
}

func TestDescribe_WithoutConfig(t *testing.T) {
	got := describe(fixtureWorkspace(nil))

	assert.Eq(t, got.TaskConfig, "")
	assert.Eq(t, got.Prompt, "")
}

func TestToDiff(t *testing.T) {
	got := toDiff(&workspace.DiffInfo{
		Commits: []workspace.CommitInfo{{Hash: "abc1234", Message: "add feature"}},
		Stat:    " feature.go | 1 +",
		Diff:    "+package main",
	})

	assert.Eq(t, len(got.Commits), 1)
	assert.Eq(t, got.Commits[0], data.Commit{Hash: "abc1234", Message: "add feature"})
	assert.Eq(t, got.Stat, " feature.go | 1 +")
	assert.Eq(t, got.Patch, "+package main")
}

func TestNotFound(t *testing.T) {
	other := errors.New("disk on fire")
	tests := []struct {
		name string
		in   error
		want error
	}{
		{"state_not_found", &state.NotFoundError{ID: "x"}, data.ErrRecordNotFound},
		{"other_error_kept", other, other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrVal(t, notFound(tt.in)).Is(tt.want).Fatal()
		})
	}
}

func TestEmitEvents_Fixtures(t *testing.T) {
	for _, name := range []string{"sample.jsonl", "faulty.jsonl"} {
		t.Run(name, func(t *testing.T) {
			lines := schematest.Lines(t, name)
			var got []data.LogEvent

			err := emitEvents(strings.NewReader(strings.Join(lines, "\n")), func(ev data.LogEvent) error {
				got = append(got, ev)
				return nil
			})

			assert.ErrVal(t, err).Nil().Fatal()
			assert.True(t, len(got) > 0)
			for _, ev := range got {
				assert.NotZero(t, ev.Kind)
			}
		})
	}
}

func TestEmitEvents_StopsWhenEmitFails(t *testing.T) {
	calls := 0

	err := emitEvents(strings.NewReader("[10:00:00] one\n[10:00:01] two\n"), func(data.LogEvent) error {
		calls++
		return errStop
	})

	assert.ErrVal(t, err).Is(errStop).Fatal()
	assert.Eq(t, calls, 1)
}

func TestLookup_FallsBackOnlyOnNotFound(t *testing.T) {
	byID := fixtureWorkspace(nil)
	byName := fixtureWorkspace(nil)
	broken := errors.New("broken state file")
	missing := &state.NotFoundError{ID: "x"}
	wrapped := fmt.Errorf("load state: %w", missing)

	tests := []struct {
		name    string
		getID   func(string) (*workspace.Workspace, error)
		getName func(string) (*workspace.Workspace, error)
		want    *workspace.Workspace
		wantErr error
	}{
		{
			"by_id",
			func(string) (*workspace.Workspace, error) { return byID, nil },
			func(string) (*workspace.Workspace, error) { return nil, broken },
			byID, nil,
		},
		{
			"id_not_found_then_name",
			func(string) (*workspace.Workspace, error) { return nil, wrapped },
			func(string) (*workspace.Workspace, error) { return byName, nil },
			byName, nil,
		},
		{
			"id_fails_with_other_error",
			func(string) (*workspace.Workspace, error) { return nil, broken },
			func(string) (*workspace.Workspace, error) { return byName, nil },
			nil, broken,
		},
		{
			"both_not_found",
			func(string) (*workspace.Workspace, error) { return nil, wrapped },
			func(string) (*workspace.Workspace, error) { return nil, missing },
			nil, data.ErrRecordNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lookup("x", tt.getID, tt.getName)

			if tt.wantErr != nil {
				assert.ErrVal(t, err).Is(tt.wantErr).Fatal()
			} else {
				assert.ErrVal(t, err).Nil().Fatal()
			}
			assert.True(t, got == tt.want)
		})
	}
}

func TestInWorkDir(t *testing.T) {
	workDir := t.TempDir()
	outside := t.TempDir()
	mustWrite := func(path string) {
		t.Helper()
		assert.ErrVal(t, os.MkdirAll(filepath.Dir(path), 0o755)).Nil().Fatal()
		assert.ErrVal(t, os.WriteFile(path, []byte("x"), 0o644)).Nil().Fatal()
	}
	mustWrite(filepath.Join(workDir, "prompt.md"))
	mustWrite(filepath.Join(workDir, "docs", "prompt.md"))
	mustWrite(filepath.Join(outside, "secret.md"))
	assert.ErrVal(t, os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(workDir, "link_out.md"))).Nil().Fatal()
	assert.ErrVal(t, os.Symlink(outside, filepath.Join(workDir, "dir_out"))).Nil().Fatal()
	assert.ErrVal(t, os.Symlink(filepath.Join(workDir, "docs", "prompt.md"), filepath.Join(workDir, "link_in.md"))).Nil().Fatal()

	tests := []struct {
		name   string
		file   string
		wantOK bool
	}{
		{"plain", "prompt.md", true},
		{"nested", "docs/prompt.md", true},
		{"parent", "../outside.md", false},
		{"nested_parent", "docs/../../outside.md", false},
		{"symlink_outside", "link_out.md", false},
		{"symlink_dir_outside", "dir_out/secret.md", false},
		{"symlink_inside", "link_in.md", true},
		{"missing", "absent.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := inWorkDir(workDir, tt.file)

			assert.Eq(t, ok, tt.wantOK)
		})
	}
}
