// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestLoadTaskConfig(t *testing.T) {
	task, err := LoadTaskConfig("testdata/task_valid.yaml", NewGlobal())
	assert.NotErr(t, err)
	assert.Equal(t, task.Resources.CPU, DefaultCPUs)
	assert.Equal(t, task.Resources.Memory, "16G")
}

func TestLoadGlobalConfig(t *testing.T) {
	cfg, err := LoadGlobalConfig("testdata/global")
	assert.NotErr(t, err)
	assert.Equal(t, cfg.Resources.CPU, "16")
}

func TestLoadGlobalConfig_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := LoadGlobalConfig(t.TempDir())
	assert.NotErr(t, err)
	assert.Equal(t, cfg.Agent.MaxTurns, DefaultMaxTurns)
	assert.Equal(t, cfg.Resources.CPU, DefaultCPUs)
	assert.Equal(t, cfg.Resources.Memory, DefaultMemory)
}

func TestLoadTaskConfig_InheritsGlobalSections(t *testing.T) {
	global, err := LoadGlobalConfig("testdata/global")
	assert.NotErr(t, err)

	task, err := LoadTaskConfig("testdata/task_valid.yaml", global)
	assert.NotErr(t, err)

	// The task provides agent and resources, so those win. It omits git and proxy,
	// which are inherited as whole sections from global. The default provider's
	// domains are added to the proxy allowlist.
	assert.Equal(t, task.Agent.MaxTurns, 9999)
	assert.Equal(t, task.Resources.Memory, "16G")
	assert.Equal(t, task.Git.Name, "Global User")
	assert.Equal(t, task.Git.Email, "global@test.com")
	assert.Equal(t, task.Network.Proxy.Port, 3128)
	assert.Equal(t, slices.Contains(task.Network.Proxy.AllowList, ".anthropic.com"), true)
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
	config.Agent.Provider.Model = "test_model"
	assert.NotErr(t, SaveGlobalConfig(tmpDir, &config))
}

func TestInitConfig_CreatesFile(t *testing.T) {
	dir := t.TempDir()

	path, err := InitConfig(dir)
	assert.NotErr(t, err)
	assert.Equal(t, path, filepath.Join(dir, "config.yaml"))

	cfg, errLoad := LoadGlobalConfig(dir)
	assert.NotErr(t, errLoad)
	assert.Equal(t, cfg.Agent.MaxTurns, DefaultMaxTurns)
	assert.Equal(t, cfg.Resources.CPU, DefaultCPUs)
}

func TestInitConfig_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	original := &Global{}
	original.Agent.Provider.Model = "custom-model"
	assert.NotErr(t, SaveGlobalConfig(dir, original))

	path, err := InitConfig(dir)
	assert.ErrIs(t, err, ErrConfigExists)
	assert.Equal(t, path, filepath.Join(dir, "config.yaml"))

	cfg, errLoad := LoadGlobalConfig(dir)
	assert.NotErr(t, errLoad)
	assert.Equal(t, cfg.Agent.Provider.Model, "custom-model")
}

func TestInitConfig_PopulatesGitIdentity(t *testing.T) {
	dir := t.TempDir()

	_, err := InitConfig(dir)
	assert.NotErr(t, err)

	cfg, errLoad := LoadGlobalConfig(dir)
	assert.NotErr(t, errLoad)

	name, email := gitGlobalIdentity()
	assert.Equal(t, cfg.Git.Name, name)
	assert.Equal(t, cfg.Git.Email, email)
}
