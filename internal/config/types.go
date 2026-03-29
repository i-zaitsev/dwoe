// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"errors"
	"path/filepath"
)

// Task defines the configuration for running a Claude agent in an isolated container.
// It is typically loaded from a task.yaml file via LoadTaskConfig or LoadMergedConfig.
//
// A Task specifies:
//   - Source: where to get the code (git repo or local path), plus prompt/spec files
//   - Agent: model selection, turn limits, environment variables, and permissions
//   - Network: proxy settings and domain allowlists for filtered internet access
//   - Resources: CPU and memory limits for the container
//
// Use Validate to check required fields and ApplyDefaults to fill in missing values.
type Task struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description,omitempty"`
	Source      Source    `yaml:"source"`
	Agent       Agent     `yaml:"agent,omitempty"`
	Git         GitUser   `yaml:"git,omitempty"`
	Network     Network   `yaml:"network,omitempty"`
	Resources   Resources `yaml:"resources,omitempty"`
	NoProxy     bool      `yaml:"no_proxy,omitempty"`
}

// Validate checks that the task configuration is valid.
// It requires a non-empty name and a valid source configuration.
func (t *Task) Validate() error {
	return t.Source.Validate()
}

// ApplyDefaults fills in zero-valued fields with package-level defaults.
// It sets default values for an agent model, max turns, CPU, and memory.
func (t *Task) ApplyDefaults() *Task {
	if t.Agent.Image == "" {
		if t.Agent.Language != "" {
			t.Agent.Image = "dwoe-agent:" + t.Agent.Language
		} else {
			t.Agent.Image = DefaultImage
		}
	}
	if t.Agent.MaxTurns == 0 {
		t.Agent.MaxTurns = DefaultMaxTurns
	}
	if t.Agent.Model == "" {
		t.Agent.Model = DefaultModel
	}
	if len(t.Agent.Permissions) == 0 {
		t.Agent.Permissions = DefaultPermissions
	}
	if !t.NoProxy && len(t.Network.Proxy.AllowList) == 0 {
		t.Network.Proxy.AllowList = DefaultAllowList
	}
	if t.Resources.CPU == "" {
		t.Resources.CPU = DefaultCPUs
	}
	if t.Resources.Memory == "" {
		t.Resources.Memory = DefaultMemory
	}
	return t
}

// ResolvePaths converts relative paths in Source fields to absolute paths
// based on the given task directory. Paths that are already absolute are unchanged.
func (t *Task) ResolvePaths(taskDir string) {
	taskDirAbs, err := filepath.Abs(taskDir)
	if err != nil {
		taskDirAbs = taskDir
	}

	files := []*string{&t.Source.PromptFile, &t.Source.SpecFile, &t.Source.LocalPath, &t.Network.AllowListFile}
	for i := range files {
		if file := files[i]; *file != "" {
			if !filepath.IsAbs(*file) {
				*file = filepath.Join(taskDirAbs, *file)
			}
			*file = filepath.Clean(*file)
		}
	}
}

// Source specifies where the task's code comes from.
// Either Repo or LocalPath must be set, but not both.
type Source struct {
	Repo       string `yaml:"repo,omitempty"`
	LocalPath  string `yaml:"local_path,omitempty"`
	Branch     string `yaml:"branch,omitempty"`
	PromptFile string `yaml:"prompt_file,omitempty"`
	SpecFile   string `yaml:"spec_file,omitempty"`
}

// Validate checks that the source configuration is valid.
// Either Repo or LocalPath must be set, but not both.
// If Repo is set, Branch is required.
func (s *Source) Validate() error {
	if s.Repo == "" && s.LocalPath == "" {
		return errors.New("either repo or local path should be provided")
	}
	if s.Repo != "" && s.LocalPath != "" {
		return errors.New("repo and local path cannot be used together")
	}
	if s.Repo != "" && s.Branch == "" {
		return errors.New("repo branch is not provided")
	}
	return nil
}

// Agent configures the Claude agent that executes the task.
// It includes model selection, turn limits, environment variables,
// and permission settings.
type Agent struct {
	Image       string            `yaml:"image,omitempty"`
	Language    string            `yaml:"language,omitempty"`
	Model       string            `yaml:"model,omitempty"`
	MaxTurns    int               `yaml:"max_turns,omitempty"`
	TaskPrompt  string            `yaml:"task_prompt,omitempty"`
	EnvVars     map[string]string `yaml:"env_vars,omitempty"`
	Permissions []string          `yaml:"permissions,omitempty"`
}

// Resources defines CPU and memory limits for the task container.
type Resources struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// Global represents the user's global configuration stored in the data directory.
// It provides default values for tasks and settings that apply across all tasks.
type Global struct {
	Defaults struct {
		Agent     Agent     `yaml:"agent,omitempty"`
		Resources Resources `yaml:"resources,omitempty"`
	} `yaml:"defaults,omitempty"`
	GitUser GitUser `yaml:"git,omitempty"`
	Proxy   Proxy   `yaml:"proxy,omitempty"`
}

// GlobalWithDefaults returns a new Global configuration with all default values set.
func GlobalWithDefaults() *Global {
	var g Global
	g.Defaults.Agent.Model = DefaultModel
	g.Defaults.Agent.MaxTurns = DefaultMaxTurns
	g.Defaults.Resources.CPU = DefaultCPUs
	g.Defaults.Resources.Memory = DefaultMemory
	return &g
}

// GitUser holds git identity configuration for commits made during task execution.
type GitUser struct {
	Name  string `yaml:"user_name,omitempty"`
	Email string `yaml:"user_email,omitempty"`
}

// Proxy configures the network proxy used to filter task network access.
type Proxy struct {
	Image     string   `yaml:"image,omitempty"`
	Port      int      `yaml:"port,omitempty"`
	AllowList []string `yaml:"base_allowlist,omitempty"`
}

// Network configures network access policies for a task.
// It can reference a shared proxy configuration and specify additional allowed domains.
type Network struct {
	Proxy          Proxy    `yaml:"proxy,omitempty"`
	AllowListFile  string   `yaml:"allowlist_file,omitempty"`
	AllowListExtra []string `yaml:"allowlist_extra,omitempty"`
	Name           string   `yaml:"name,omitempty"`
	Subnet         string   `yaml:"subnet,omitempty"`
	Gateway        string   `yaml:"gateway,omitempty"`
}
