// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"gopkg.in/yaml.v3"
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

func TestTask_FallbackSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		task     Task
		dir      string
		wantPath string
	}{
		{"sets_when_empty", Task{}, "/fallback", "/fallback"},
		{"noop_when_local_path_set", Task{Source: Source{LocalPath: "/existing"}}, "/fallback", "/existing"},
		{"noop_when_repo_set", Task{Source: Source{Repo: "org/repo", Branch: "main"}}, "/fallback", ""},
		{"noop_when_dir_empty", Task{}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.task.FallbackSource(tt.dir)
			assert.Equal(t, tt.task.Source.LocalPath, tt.wantPath)
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

func TestContinuePolicy_UnmarshalYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		yaml    string
		want    ContinuePolicy
		wantErr bool
	}{
		{`continue_policy: ""`, ContinuePolicyDefault, false},
		{`continue_policy: default`, ContinuePolicyDefault, false},
		{`continue_policy: restart`, ContinuePolicyRestart, false},
		{`continue_policy: resume`, ContinuePolicyResume, false},
		{`continue_policy: bogus`, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.yaml, func(t *testing.T) {
			t.Parallel()
			var task Task
			err := yaml.Unmarshal([]byte(tt.yaml), &task)
			if tt.wantErr {
				assert.Err(t, err)
				return
			}
			assert.NotErr(t, err)
			assert.Equal(t, task.ContinuePolicy, tt.want)
		})
	}
}

func TestTask_PolicyRequiresNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		policy ContinuePolicy
		want   bool
	}{
		{ContinuePolicyDefault, true},
		{ContinuePolicyRestart, true},
		{ContinuePolicyResume, false},
	}
	for _, tt := range tests {
		task := &Task{ContinuePolicy: tt.policy}
		assert.Equal(t, task.PolicyRequiresNew(), tt.want)
	}
}
