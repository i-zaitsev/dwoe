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
	"github.com/i-zaitsev/pkg/assert"
)

func TestLoadTaskConfig(t *testing.T) {
	task, err := LoadTaskConfig("testdata/task_valid.yaml", NewGlobal())
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, task.Resources.CPU, DefaultCPUs)
	assert.Eq(t, task.Resources.Memory, "16G")
}

func TestLoadGlobalConfig(t *testing.T) {
	cfg, err := LoadGlobalConfig("testdata/global")
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, cfg.Resources.CPU, "16")
}

func TestLoadGlobalConfig_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := LoadGlobalConfig(t.TempDir())
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, cfg.Agent.MaxTurns, DefaultMaxTurns)
	assert.Eq(t, cfg.Resources.CPU, DefaultCPUs)
	assert.Eq(t, cfg.Resources.Memory, DefaultMemory)
}

func TestLoadTaskConfig_InheritsGlobalSections(t *testing.T) {
	global, err := LoadGlobalConfig("testdata/global")
	assert.ErrVal(t, err).Nil().Fatal()

	task, err := LoadTaskConfig("testdata/task_valid.yaml", global)
	assert.ErrVal(t, err).Nil().Fatal()

	// The task provides agent and resources, so those win. It omits git and proxy,
	// which are inherited as whole sections from global. The default provider's
	// domains are added to the proxy allowlist.
	assert.Eq(t, task.Agent.MaxTurns, 9999)
	assert.Eq(t, task.Resources.Memory, "16G")
	assert.Eq(t, task.Git.Name, "Global User")
	assert.Eq(t, task.Git.Email, "global@test.com")
	assert.Eq(t, task.Network.Proxy.Port, 3128)
	assert.Eq(t, slices.Contains(task.Network.Proxy.AllowList, ".anthropic.com"), true)
}

func TestLoadAllowListFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.txt")
	content := "example.com\n# comment\n\n  *.go.dev  \nnpmjs.org\n"
	assert.ErrVal(t, os.WriteFile(path, []byte(content), 0o644)).Nil().Fatal()

	got, err := loadAllowListFile(path)
	assert.ErrVal(t, err).Nil().Fatal()
	want := []string{"example.com", "*.go.dev", "npmjs.org"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
}

func TestSaveGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	var config Global
	config.Agent.Provider.Model = "test_model"
	assert.ErrVal(t, SaveGlobalConfig(tmpDir, &config)).Nil().Fatal()
}

func TestInitConfig_CreatesFile(t *testing.T) {
	dir := t.TempDir()

	path, err := InitConfig(dir)
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, path, filepath.Join(dir, "config.yaml"))

	cfg, errLoad := LoadGlobalConfig(dir)
	assert.ErrVal(t, errLoad).Nil().Fatal()
	assert.Eq(t, cfg.Agent.MaxTurns, DefaultMaxTurns)
	assert.Eq(t, cfg.Resources.CPU, DefaultCPUs)
}

func TestInitConfig_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	original := &Global{}
	original.Agent.Provider.Model = "custom-model"
	assert.ErrVal(t, SaveGlobalConfig(dir, original)).Nil().Fatal()

	path, err := InitConfig(dir)
	assert.ErrVal(t, err).Is(ErrConfigExists).Fatal()
	assert.Eq(t, path, filepath.Join(dir, "config.yaml"))

	cfg, errLoad := LoadGlobalConfig(dir)
	assert.ErrVal(t, errLoad).Nil().Fatal()
	assert.Eq(t, cfg.Agent.Provider.Model, "custom-model")
}

func TestInitConfig_PopulatesGitIdentity(t *testing.T) {
	dir := t.TempDir()

	_, err := InitConfig(dir)
	assert.ErrVal(t, err).Nil().Fatal()

	cfg, errLoad := LoadGlobalConfig(dir)
	assert.ErrVal(t, errLoad).Nil().Fatal()

	name, email := gitGlobalIdentity()
	assert.Eq(t, cfg.Git.Name, name)
	assert.Eq(t, cfg.Git.Email, email)
}
