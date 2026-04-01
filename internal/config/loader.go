// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrConfigExists = errors.New("config already exists")

// LoadTaskConfig loads a task configuration from a YAML file at the given path.
// It parses the file, resolves relative paths against the task file's directory,
// and validates the configuration. Returns an error if the file cannot be read,
// parsed, or if validation fails.
func LoadTaskConfig(path string) (*Task, error) {
	slog.Info("config: load-task", "path", path)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %s: %w", path, err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	var task Task
	if errDec := decoder.Decode(&task); errDec != nil {
		return nil, fmt.Errorf("decode YAML: %w", errDec)
	}

	task.ResolvePaths(filepath.Dir(path))

	if task.Network.AllowListFile != "" {
		slog.Debug("config: load-task", "allowlist", task.Network.AllowListFile)
		extra, errAL := loadAllowListFile(task.Network.AllowListFile)
		if errAL != nil {
			return nil, fmt.Errorf("load allowlist file: %w", errAL)
		}
		task.Network.Proxy.AllowList = append(task.Network.Proxy.AllowList, extra...)
	}

	return &task, nil
}

// LoadGlobalConfig loads global configuration from config.yaml in the given data directory.
// If the file does not exist, it returns a Global with default values.
// Returns an error only if the file exists but cannot be read or parsed.
func LoadGlobalConfig(dataDir string) (*Global, error) {
	slog.Debug("config: load-global", "datadir", dataDir)
	configPath := filepath.Join(dataDir, "config.yaml")

	f, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("config: load-global", "message", "config file not found, using defaults")
			return GlobalWithDefaults(), nil
		}
		return nil, fmt.Errorf("cannot open global config %s: %w", configPath, err)
	}
	defer func() {
		_ = f.Close()
	}()

	slog.Debug("config: load-global", "message", "reading global config")
	decoder := yaml.NewDecoder(f)
	var global Global
	if errDec := decoder.Decode(&global); errDec != nil {
		return nil, fmt.Errorf("cannot decode YAML: %w", errDec)
	}

	return &global, nil
}

// LoadMergedConfig loads a task configuration and merges it with global defaults.
// Global config values are applied only where the task config has no value set.
// After merging, ApplyDefaults is called to fill any remaining empty fields.
func LoadMergedConfig(taskPath, dataDir string) (*Task, error) {
	slog.Info("config: load-merged", "task", taskPath, "datadir", dataDir)
	var err error

	global, err := LoadGlobalConfig(dataDir)
	if err != nil {
		return nil, fmt.Errorf("cannot load global config: %w", err)
	}

	task, err := LoadTaskConfig(taskPath)
	if err != nil {
		return nil, fmt.Errorf("cannot load task config: %w", err)
	}

	slog.Debug("config: load-merged", "message", "merging task config with global defaults")
	MergeWithGlobal(task, global)
	task.ApplyDefaults()

	return task, nil
}

// MergeWithGlobal merges task configuration with global defaults.
func MergeWithGlobal(task *Task, global *Global) {
	task.Agent.Model = ifZero(task.Agent.Model, global.Defaults.Agent.Model)
	task.Agent.MaxTurns = ifZero(task.Agent.MaxTurns, global.Defaults.Agent.MaxTurns)
	task.Resources.CPU = ifZero(task.Resources.CPU, global.Defaults.Resources.CPU)
	task.Resources.Memory = ifZero(task.Resources.Memory, global.Defaults.Resources.Memory)
	task.Git.Name = ifZero(task.Git.Name, global.GitUser.Name)
	task.Git.Email = ifZero(task.Git.Email, global.GitUser.Email)
	task.Network.Proxy.Port = ifZero(task.Network.Proxy.Port, global.Proxy.Port)
	task.Network.Proxy.AllowList = ifEmpty(task.Network.Proxy.AllowList, global.Proxy.AllowList)
}

// SaveGlobalConfig writes the global configuration to config.yaml in the given data directory.
// It creates the directory if it does not exist. Returns an error if the directory
// cannot be created or the file cannot be written.
func SaveGlobalConfig(dataDir string, config *Global) error {
	slog.Info("config: save-global", "datadir", dataDir)
	if err := os.MkdirAll(dataDir, 0o775); err != nil {
		return fmt.Errorf("cannot create data dir: %s: %w", dataDir, err)
	}

	configPath := filepath.Join(dataDir, "config.yaml")
	slog.Debug("config: save-global", "path", configPath)
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("cannot create config file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	slog.Debug("config: save-global", "message", "encoding global config to YAML")
	encoder := yaml.NewEncoder(f)
	if errEnc := encoder.Encode(config); errEnc != nil {
		return fmt.Errorf("cannot encode config: %w", errEnc)
	}

	return nil
}

// loadAllowListFile reads a file containing a list of domains, one per line.
// It ignores empty lines and lines starting with "#".
func loadAllowListFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	slog.Debug("config: load-allowlist", "path", path)
	var domains []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		domains = append(domains, line)
	}

	slog.Debug("config: load-allowlist", "count", len(domains))
	return domains, scanner.Err()
}

// InitConfig creates a default config.yaml in dataDir if it does not exist.
// Always returns the config file path.
// Returns nil error on creation, ErrConfigExists if the file already exists,
// or another error if creation fails.
func InitConfig(dataDir string) (string, error) {
	configPath := filepath.Join(dataDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, ErrConfigExists
	}
	cfg := GlobalWithDefaults()
	name, email := gitGlobalIdentity()
	cfg.GitUser.Name = name
	cfg.GitUser.Email = email
	if err := SaveGlobalConfig(dataDir, cfg); err != nil {
		return configPath, err
	}
	return configPath, nil
}

func gitGlobalIdentity() (string, string) {
	name, _ := exec.Command("git", "config", "--global", "user.name").Output()
	email, _ := exec.Command("git", "config", "--global", "user.email").Output()
	return strings.TrimSpace(string(name)), strings.TrimSpace(string(email))
}

// ifZero returns val if it is not zero, otherwise it returns fallback.
func ifZero[T comparable](val, fallback T) T {
	var zero T
	if val == zero {
		return fallback
	}
	return val
}

// ifEmpty returns val if it is not empty, otherwise it returns fallback.
// Works with slices and maps.
func ifEmpty[S ~[]T, T any](val, fallback S) S {
	if len(val) == 0 {
		return fallback
	}
	return val
}
