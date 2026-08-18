// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config/provider"
)

func TestMain(m *testing.M) {
	provider.RegisterDefaultProviders()
	os.Exit(m.Run())
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

	build := func(mutate func(*Task)) *Task {
		task := &Task{
			Name:      "test",
			Source:    Source{Repo: "example/repo", Branch: "main"},
			Agent:     &Agent{Provider: provider.Lookup(provider.DefaultModelProvider)},
			Resources: NewResources(),
			Git:       &GitUser{},
		}
		if mutate != nil {
			mutate(task)
		}
		return task
	}

	tests := []struct {
		name    string
		task    *Task
		wantErr string
	}{
		{
			"valid task",
			build(nil),
			"",
		},
		{
			"empty name allowed",
			build(func(t *Task) { t.Name = "" }),
			"",
		},
		{
			"invalid source propagates",
			build(func(t *Task) { t.Source = Source{} }),
			"either repo or local path should be provided",
		},
		{
			"missing agent",
			build(func(t *Task) { t.Agent = nil }),
			"agent is not configured",
		},
		{
			"missing resources",
			build(func(t *Task) { t.Resources = nil }),
			"resources are not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkErr(t, tt.task.Validate(), tt.wantErr)
		})
	}
}

func TestWithFallbackSourceDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		source   Source
		dir      string
		wantPath string
	}{
		{"sets_when_empty", Source{}, "/fallback", "/fallback"},
		{"noop_when_local_path_set", Source{LocalPath: "/existing"}, "/fallback", "/existing"},
		{"noop_when_repo_set", Source{Repo: "org/repo", Branch: "main"}, "/fallback", ""},
		{"noop_when_dir_empty", Source{}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			task := NewTask("test", func(task *Task) { task.Source = tt.source }, WithFallbackSourceDir(tt.dir))
			assert.Equal(t, task.Source.LocalPath, tt.wantPath)
		})
	}
}

func TestWithDefaults(t *testing.T) {
	t.Parallel()

	task := NewTask("test", WithDefaults())

	assert.Equal(t, task.Agent.MaxTurns, DefaultMaxTurns)
	assert.Equal(t, task.Resources.CPU, DefaultCPUs)
	assert.Equal(t, task.Resources.Memory, DefaultMemory)
	assert.Equal(t, task.Agent.Image, DefaultImage)
	assert.Equal(t, len(task.Agent.Permissions), len(DefaultPermissions))
}

func TestWithModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fileModel string
		flagModel string
		want      string
	}{
		{"flag overrides file", "file-model", "flag-model", "flag-model"},
		{"empty flag keeps file", "file-model", "", "file-model"},
		{"flag used when file empty", "", "flag-model", "flag-model"},
		{"both empty fall back to provider", "", "", provider.DefaultAnthropicModel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			task := NewTask("test", func(task *Task) {
				task.Agent = &Agent{Provider: provider.Provider{Model: tt.fileModel}}
			}, WithModel(tt.flagModel), WithDefaults())

			assert.Equal(t, task.Agent.Provider.Model, tt.want)
		})
	}
}

func TestWithDefaults_LanguageImage(t *testing.T) {
	t.Parallel()

	task := NewTask("test", func(task *Task) { task.Agent = &Agent{Language: "go"} }, WithDefaults())

	assert.Equal(t, task.Agent.Image, "dwoe-agent:go")
}

func TestWithDefaults_ExplicitPermissions(t *testing.T) {
	t.Parallel()

	custom := []string{"Bash(go:*)"}
	task := NewTask("test", func(task *Task) { task.Agent = &Agent{Permissions: custom} }, WithDefaults())

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
