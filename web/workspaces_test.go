// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"errors"
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

func TestInWorkDir(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		wantOK bool
	}{
		{"plain", "prompt.md", true},
		{"nested", "docs/prompt.md", true},
		{"parent", "../outside.md", false},
		{"nested_parent", "docs/../../outside.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := inWorkDir("testdata/ws-alpha/workspace", tt.file)

			assert.Eq(t, ok, tt.wantOK)
		})
	}
}
