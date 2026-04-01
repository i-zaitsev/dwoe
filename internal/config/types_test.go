// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func checkErr(t *testing.T, err error, wantErr string) {
	t.Helper()
	if wantErr == "" {
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	} else {
		if err == nil {
			t.Errorf("expected error %q, got nil", wantErr)
		} else if err.Error() != wantErr {
			t.Errorf("expected error %q, got %q", wantErr, err.Error())
		}
	}
}

func TestSource_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  Source
		wantErr string
	}{
		{
			name:    "valid repo source",
			source:  Source{Repo: "example/repo", Branch: "main"},
			wantErr: "",
		},
		{
			name:    "valid local path source",
			source:  Source{LocalPath: "/path/to/local"},
			wantErr: "",
		},
		{
			name:    "missing source",
			source:  Source{},
			wantErr: "either repo or local path should be provided",
		},
		{
			name:    "both repo and local path",
			source:  Source{Repo: "example/repo", LocalPath: "/path/to/local", Branch: "main"},
			wantErr: "repo and local path cannot be used together",
		},
		{
			name:    "repo without branch",
			source:  Source{Repo: "example/repo"},
			wantErr: "repo branch is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkErr(t, tt.source.Validate(), tt.wantErr)
		})
	}
}

func TestTask_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		task    Task
		wantErr string
	}{
		{
			name: "valid task",
			task: Task{
				Name:   "test",
				Source: Source{Repo: "example/repo", Branch: "main"},
			},
			wantErr: "",
		},
		{
			name: "empty name",
			task: Task{
				Name:   "",
				Source: Source{Repo: "example/repo", Branch: "main"},
			},
			wantErr: "",
		},
		{
			name: "invalid source propagates",
			task: Task{
				Name:   "test",
				Source: Source{},
			},
			wantErr: "either repo or local path should be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkErr(t, tt.task.Validate(), tt.wantErr)
		})
	}
}

func TestTask_ApplyDefaults(t *testing.T) {
	t.Parallel()

	task := &Task{}
	task.ApplyDefaults()

	testFields := []struct {
		want, got string
	}{
		{DefaultImage, task.Agent.Image},
		{fmt.Sprintf("%d", DefaultMaxTurns), fmt.Sprintf("%d", task.Agent.MaxTurns)},
		{DefaultModel, task.Agent.Model},
		{DefaultCPUs, task.Resources.CPU},
		{DefaultMemory, task.Resources.Memory},
	}

	for _, tc := range testFields {
		assert.Equal(t, tc.want, tc.got)
	}

	assert.Equal(t, len(task.Agent.Permissions), len(DefaultPermissions))
}

func TestTask_ApplyDefaults_LanguageImage(t *testing.T) {
	t.Parallel()

	task := &Task{Agent: Agent{Language: "go"}}
	task.ApplyDefaults()

	assert.Equal(t, task.Agent.Image, "dwoe-agent:go")
}

func TestTask_ApplyDefaults_ExplicitPermissions(t *testing.T) {
	t.Parallel()

	custom := []string{"Bash(go:*)"}
	task := &Task{Agent: Agent{Permissions: custom}}
	task.ApplyDefaults()

	if len(task.Agent.Permissions) != 1 || task.Agent.Permissions[0] != "Bash(go:*)" {
		t.Errorf("explicit permissions should not be overridden, got %v", task.Agent.Permissions)
	}
}

func TestTask_ResolvePaths(t *testing.T) {
	t.Parallel()

	task := &Task{}
	task.Source.SpecFile = "spec.md"
	task.Source.PromptFile = "prompt.txt"
	testDir := t.TempDir()

	task.ResolvePaths(testDir)

	for _, tc := range []struct {
		want, got string
	}{
		{filepath.Join(testDir, "spec.md"), task.Source.SpecFile},
		{filepath.Join(testDir, "prompt.txt"), task.Source.PromptFile},
	} {
		assert.Equal(t, tc.want, tc.got)
	}
}
