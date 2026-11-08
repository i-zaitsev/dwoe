// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/testutil"
	"github.com/i-zaitsev/dwoe/internal/workspace"
	"github.com/i-zaitsev/pkg/assert"
)

func TestResolvePromptContent(t *testing.T) {
	base := t.TempDir()
	workDir := filepath.Join(base, "workspace")
	testutil.WriteFile(t, filepath.Join(workDir, "prompt.md"), "file content")

	tests := []struct {
		name       string
		taskPrompt string
		promptFile string
		want       string
	}{
		{"inline_prompt_wins", "inline body", "prompt.md", "inline body"},
		{"reads_file", "", "prompt.md", "file content"},
		{"missing_file", "", "missing.md", ""},
		{"empty_when_none", "", "", ""},
		{"absolute_rejected", "", "/etc/passwd", ""},
		{"traversal_rejected", "", "../outside.md", ""},
		{"nested_traversal_rejected", "", "sub/../../outside.md", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := workspace.New(
				state.EmptyWorkspace("test-id", "test-ws"),
				&config.Task{
					Agent:  &config.Agent{TaskPrompt: tt.taskPrompt},
					Source: config.Source{PromptFile: tt.promptFile},
				},
			)
			ws.BasePath = base
			assert.Eq(t, resolvePromptContent(ws), tt.want)
		})
	}
}
