// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadTaskConfig(t *testing.T) {
	path := "testdata/task_valid.yaml"
	task, err := LoadTaskConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if task.Resources.CPU != "" {
		t.Errorf("Expected CPU to be empty (not merged), got '%s'", task.Resources.CPU)
	}
	if task.Resources.Memory != "16G" {
		t.Errorf("Expected memory to be '16G', got '%s'", task.Resources.Memory)
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	globalDir := "testdata/global"
	config, err := LoadGlobalConfig(globalDir)
	if err != nil {
		t.Fatal(err)
	}
	want := "16"
	if got := config.Defaults.Resources.CPU; got != want {
		t.Errorf("want %s\ngot  %s", want, got)
	}
}

func TestLoadGlobalConfig_MissingFileReturnsDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	config, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("expected no error loading default config, got %v", err)
	}
	testCases := []struct {
		want string
		got  string
	}{
		{config.Defaults.Agent.Model, DefaultModel},
		{fmt.Sprintf("%d", config.Defaults.Agent.MaxTurns), fmt.Sprintf("%d", DefaultMaxTurns)},
		{config.Defaults.Resources.CPU, DefaultCPUs},
		{config.Defaults.Resources.Memory, DefaultMemory},
	}
	for _, tc := range testCases {
		if tc.want != tc.got {
			t.Errorf("want %s\ngot  %s", tc.want, tc.got)
		}
	}
}

func TestLoadMergedConfig(t *testing.T) {
	taskPath := "testdata/task_valid.yaml"
	globalDir := "testdata/global"

	cfg, err := LoadMergedConfig(taskPath, globalDir)

	if err != nil {
		t.Fatal(err)
	}
	testFields := []struct {
		got, want string
	}{
		{cfg.Resources.CPU, "16"},
		{cfg.Resources.Memory, "16G"},
		{cfg.Agent.Model, "test-model"},
		{fmt.Sprintf("%d", cfg.Agent.MaxTurns), fmt.Sprintf("%d", 9999)},
	}
	for _, tc := range testFields {
		if tc.want != tc.got {
			t.Errorf("want %s\ngot  %s", tc.want, tc.got)
		}
	}
}

func TestLoadMergedConfig_GitAndProxy(t *testing.T) {
	taskPath := "testdata/task_valid.yaml"
	globalDir := "testdata/global"

	cfg, err := LoadMergedConfig(taskPath, globalDir)

	if err != nil {
		t.Fatal(err)
	}
	testFields := []struct {
		got, want string
	}{
		{cfg.Git.Name, "Global User"},
		{cfg.Git.Email, "global@test.com"},
		{fmt.Sprintf("%d", cfg.Network.Proxy.Port), fmt.Sprintf("%d", 3128)},
		{fmt.Sprintf("%d", len(cfg.Network.Proxy.AllowList)), fmt.Sprintf("%d", 2)},
		{cfg.Agent.Image, DefaultImage},
	}
	for _, tc := range testFields {
		if tc.want != tc.got {
			t.Errorf("want %s, got '%s'", tc.want, tc.got)
		}
	}
}

func TestMergeWithGlobal(t *testing.T) {
	global := &Global{}
	global.Defaults.Agent.Model = "global-model"
	global.Defaults.Agent.MaxTurns = 100
	global.Defaults.Resources.CPU = "8"
	global.Defaults.Resources.Memory = "16G"
	global.GitUser.Name = "Global User"
	global.GitUser.Email = "global@test.com"
	global.Proxy.Port = 3128
	global.Proxy.AllowList = []string{".npmjs.org"}

	t.Run("fills_empty_fields", func(t *testing.T) {
		task := &Task{}
		MergeWithGlobal(task, global)

		fields := []struct{ got, want string }{
			{task.Agent.Model, "global-model"},
			{fmt.Sprintf("%d", task.Agent.MaxTurns), "100"},
			{task.Resources.CPU, "8"},
			{task.Resources.Memory, "16G"},
			{task.Git.Name, "Global User"},
			{task.Git.Email, "global@test.com"},
			{fmt.Sprintf("%d", task.Network.Proxy.Port), "3128"},
			{fmt.Sprintf("%d", len(task.Network.Proxy.AllowList)), "1"},
		}
		for _, f := range fields {
			if f.got != f.want {
				t.Errorf("want %s, got %s", f.want, f.got)
			}
		}
	})

	t.Run("task_takes_precedence", func(t *testing.T) {
		task := &Task{
			Agent:     Agent{Model: "task-model", MaxTurns: 50},
			Resources: Resources{CPU: "2", Memory: "4G"},
			Git:       GitUser{Name: "Task User", Email: "task@test.com"},
		}
		task.Network.Proxy.Port = 9999
		task.Network.Proxy.AllowList = []string{".custom.dev"}

		MergeWithGlobal(task, global)

		fields := []struct{ got, want string }{
			{task.Agent.Model, "task-model"},
			{fmt.Sprintf("%d", task.Agent.MaxTurns), "50"},
			{task.Resources.CPU, "2"},
			{task.Resources.Memory, "4G"},
			{task.Git.Name, "Task User"},
			{task.Git.Email, "task@test.com"},
			{fmt.Sprintf("%d", task.Network.Proxy.Port), "9999"},
			{task.Network.Proxy.AllowList[0], ".custom.dev"},
		}
		for _, f := range fields {
			if f.got != f.want {
				t.Errorf("want %s, got %s", f.want, f.got)
			}
		}
	})
}

func TestLoadAllowListFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.txt")
	content := "example.com\n# comment\n\n  *.go.dev  \nnpmjs.org\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := loadAllowListFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"example.com", "*.go.dev", "npmjs.org"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
}

func TestSaveGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	var config Global
	config.Defaults.Agent.Model = "test_model"
	if err := SaveGlobalConfig(tmpDir, &config); err != nil {
		t.Fatal(err)
	}
}
