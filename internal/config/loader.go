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

// LoadTaskConfig reads a task file.
//
// It fills gaps from global settings and defaults and returns a ready-to-use task.
// Relative source paths are resolved against the task file's directory. Any referenced allowlist file is merged in.
// Caller options are applied after the file and global values but before defaults and take precedence over both.
func LoadTaskConfig(taskPath string, g *Global, opts ...TaskOpt) (*Task, error) {
	slog.Info("config: load-task", "path", taskPath)

	parsed, err := decodeTaskFile(taskPath)
	if err != nil {
		return nil, err
	}

	task := NewTaskFrom(parsed, g, opts...)
	task.ResolvePaths(filepath.Dir(taskPath))

	if task.Network.AllowListFile != "" {
		slog.Debug("config: load-task", "allowlist", task.Network.AllowListFile)
		extra, errAL := loadAllowListFile(task.Network.AllowListFile)
		if errAL != nil {
			return nil, fmt.Errorf("load allowlist file: %w", errAL)
		}
		task.Network.Proxy.AllowList = append(task.Network.Proxy.AllowList, extra...)
	}

	return task, nil
}

// decodeTaskFile reads and YAML-decodes a task file without applying any defaults.
func decodeTaskFile(taskPath string) (*Task, error) {
	f, err := os.Open(taskPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %s: %w", taskPath, err)
	}
	defer func() {
		_ = f.Close()
	}()

	var task Task
	if errDec := yaml.NewDecoder(f).Decode(&task); errDec != nil {
		return nil, fmt.Errorf("decode YAML: %w", errDec)
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
			return NewGlobal(), nil
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
	cfg := NewGlobal()
	name, email := gitGlobalIdentity()
	cfg.Git.Name = name
	cfg.Git.Email = email
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
