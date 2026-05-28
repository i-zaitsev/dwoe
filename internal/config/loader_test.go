// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestLoadTaskConfig(t *testing.T) {
	path := "testdata/task_valid.yaml"
	task, err := LoadTaskConfig(path)
	assert.NotErr(t, err)
	assert.Zero(t, task.Resources.CPU)
	assert.Equal(t, task.Resources.Memory, "16G")
}

func TestLoadGlobalConfig(t *testing.T) {
	globalDir := "testdata/global"
	config, err := LoadGlobalConfig(globalDir)
	assert.NotErr(t, err)
	assert.Equal(t, config.Defaults.Resources.CPU, "16")
}

func TestLoadGlobalConfig_MissingFileReturnsDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	config, err := LoadGlobalConfig(tmpDir)
	assert.NotErr(t, err)
	assert.Equal(t, config.Defaults.Agent.Model, DefaultModel)
	assert.Equal(t, config.Defaults.Agent.MaxTurns, DefaultMaxTurns)
	assert.Equal(t, config.Defaults.Resources.CPU, DefaultCPUs)
	assert.Equal(t, config.Defaults.Resources.Memory, DefaultMemory)
}

func TestLoadMergedConfig(t *testing.T) {
	taskPath := "testdata/task_valid.yaml"
	globalDir := "testdata/global"

	cfg, err := LoadMergedConfig(taskPath, globalDir)
	assert.NotErr(t, err)
	assert.Equal(t, cfg.Resources.CPU, "16")
	assert.Equal(t, cfg.Resources.Memory, "16G")
	assert.Equal(t, cfg.Agent.Model, "test-model")
	assert.Equal(t, cfg.Agent.MaxTurns, 9999)
}

func TestLoadMergedConfig_GitAndProxy(t *testing.T) {
	taskPath := "testdata/task_valid.yaml"
	globalDir := "testdata/global"

	cfg, err := LoadMergedConfig(taskPath, globalDir)
	assert.NotErr(t, err)
	assert.Equal(t, cfg.Git.Name, "Global User")
	assert.Equal(t, cfg.Git.Email, "global@test.com")
	assert.Equal(t, cfg.Network.Proxy.Port, 3128)
	assert.Equal(t, len(cfg.Network.Proxy.AllowList), 2)
	assert.Equal(t, cfg.Agent.Image, DefaultImage)
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

		assert.Equal(t, task.Agent.Model, "global-model")
		assert.Equal(t, task.Agent.MaxTurns, 100)
		assert.Equal(t, task.Resources.CPU, "8")
		assert.Equal(t, task.Resources.Memory, "16G")
		assert.Equal(t, task.Git.Name, "Global User")
		assert.Equal(t, task.Git.Email, "global@test.com")
		assert.Equal(t, task.Network.Proxy.Port, 3128)
		assert.Equal(t, len(task.Network.Proxy.AllowList), 1)
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

		assert.Equal(t, task.Agent.Model, "task-model")
		assert.Equal(t, task.Agent.MaxTurns, 50)
		assert.Equal(t, task.Resources.CPU, "2")
		assert.Equal(t, task.Resources.Memory, "4G")
		assert.Equal(t, task.Git.Name, "Task User")
		assert.Equal(t, task.Git.Email, "task@test.com")
		assert.Equal(t, task.Network.Proxy.Port, 9999)
		assert.Equal(t, task.Network.Proxy.AllowList[0], ".custom.dev")
	})
}

func TestLoadAllowListFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.txt")
	content := "example.com\n# comment\n\n  *.go.dev  \nnpmjs.org\n"
	assert.NotErr(t, os.WriteFile(path, []byte(content), 0o644))

	got, err := loadAllowListFile(path)
	assert.NotErr(t, err)
	want := []string{"example.com", "*.go.dev", "npmjs.org"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
}

func TestSaveGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	var config Global
	config.Defaults.Agent.Model = "test_model"
	assert.NotErr(t, SaveGlobalConfig(tmpDir, &config))
}

func TestInitConfig_CreatesFile(t *testing.T) {
	dir := t.TempDir()

	path, err := InitConfig(dir)
	assert.NotErr(t, err)
	assert.Equal(t, path, filepath.Join(dir, "config.yaml"))

	cfg, errLoad := LoadGlobalConfig(dir)
	assert.NotErr(t, errLoad)
	assert.Equal(t, cfg.Defaults.Agent.Model, DefaultModel)
	assert.Equal(t, cfg.Defaults.Resources.CPU, DefaultCPUs)
}

func TestInitConfig_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	original := &Global{}
	original.Defaults.Agent.Model = "custom-model"
	assert.NotErr(t, SaveGlobalConfig(dir, original))

	path, err := InitConfig(dir)
	assert.ErrIs(t, err, ErrConfigExists)
	assert.Equal(t, path, filepath.Join(dir, "config.yaml"))

	cfg, errLoad := LoadGlobalConfig(dir)
	assert.NotErr(t, errLoad)
	assert.Equal(t, cfg.Defaults.Agent.Model, "custom-model")
}

func TestInitConfig_PopulatesGitIdentity(t *testing.T) {
	dir := t.TempDir()

	_, err := InitConfig(dir)
	assert.NotErr(t, err)

	cfg, errLoad := LoadGlobalConfig(dir)
	assert.NotErr(t, errLoad)

	name, email := gitGlobalIdentity()
	assert.Equal(t, cfg.GitUser.Name, name)
	assert.Equal(t, cfg.GitUser.Email, email)
}
