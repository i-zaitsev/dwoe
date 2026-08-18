// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"errors"
	"path/filepath"
	"slices"

	"github.com/i-zaitsev/dwoe/internal/config/provider"
)

// Global represents the global user's configuration stored in the data directory.
// It provides default values for tasks and settings that apply across all tasks.
type Global struct {
	Agent     Agent     `yaml:"agent,omitempty"`
	Resources Resources `yaml:"resources,omitempty"`
	Git       GitUser   `yaml:"git,omitempty"`
	Proxy     Proxy     `yaml:"proxy,omitempty"`
}

// NewGlobal returns a new Global configuration with default values set.
func NewGlobal() *Global {
	var g Global
	g.Agent.MaxTurns = DefaultMaxTurns
	g.Resources.CPU = DefaultCPUs
	g.Resources.Memory = DefaultMemory
	return &g
}

// Task defines the configuration for running a Claude agent in an isolated container.
// It is typically loaded from a task.yaml file via LoadTaskConfig or LoadMergedConfig.
//
// A Task specifies:
//   - Source: where to get the code (git repo or local path), plus prompt/spec files
//   - Agent: model selection, turn limits, environment variables, and permissions
//   - Network: proxy settings and domain allowlists for filtered internet access
//   - Resources: CPU and memory limits for the container
//
// Use Validate to check required fields.
type Task struct {
	Name           string         `yaml:"name"`
	Description    string         `yaml:"description,omitempty"`
	Source         Source         `yaml:"source"`
	ContinuePolicy ContinuePolicy `yaml:"continue_policy,omitempty"`
	Network        Network        `yaml:"network,omitempty"`
	NoProxy        bool           `yaml:"no_proxy,omitempty"`

	// if not provided, configurable from Global settings
	Agent     *Agent     `yaml:"agent,omitempty"`
	Resources *Resources `yaml:"resources,omitempty"`
	Git       *GitUser   `yaml:"git"`
}

// TaskOpt configures a newly created task.
type TaskOpt func(*Task)

// NewTask creates a task with the given name and options applied.
// Provider resolution always runs after the options, since every task needs a
// resolved provider to be runnable.
func NewTask(name string, opts ...TaskOpt) *Task {
	t := &Task{Name: name}
	for _, opt := range opts {
		opt(t)
	}
	t.resolveProvider()
	return t
}

// resolveProvider backfills the agent's provider from the registry.
// The provider's auth env vars and allowed domains are intrinsic and always
// sourced from the registry. A user-set model is preserved. The provider's
// domains are added to the proxy allowlist unless the proxy is disabled.
func (t *Task) resolveProvider() {
	if t.Agent == nil {
		t.Agent = &Agent{}
	}
	name := t.Agent.Provider.Name
	if t.Agent.Provider.Unknown() {
		name = provider.DefaultModelProvider
	}
	resolved := provider.Lookup(name)
	if t.Agent.Provider.Model != "" {
		resolved.Model = t.Agent.Provider.Model
	}
	t.Agent.Provider = resolved
	if t.NoProxy {
		return
	}
	for _, domain := range t.Agent.Provider.AllowDomains {
		if !slices.Contains(t.Network.Proxy.AllowList, domain) {
			t.Network.Proxy.AllowList = append(t.Network.Proxy.AllowList, domain)
		}
	}
}

// NewTaskWithParams creates a task configured from global settings and with default values initialized.
func NewTaskWithParams(name, sourceDir string, g *Global) *Task {
	return NewTask(name, WithGlobals(g), WithFallbackSourceDir(sourceDir), WithDefaults())
}

// WithGlobals fills task sections that are not set with values copied from global.
// Value copies each section so a task never shares state with the Global.
func WithGlobals(g *Global) TaskOpt {
	return func(t *Task) {
		if t.Agent == nil {
			agent := g.Agent
			t.Agent = &agent
		}
		if t.Resources == nil {
			resources := g.Resources
			t.Resources = &resources
		}
		if t.Git == nil {
			git := g.Git
			t.Git = &git
		}
		if !t.Network.Proxy.configured() {
			t.Network.Proxy = g.Proxy
		}
	}
}

// NewTaskFrom builds a complete task from src, filling gaps from global settings
// and defaults. Options are applied before the defaults.
func NewTaskFrom(src *Task, g *Global, opts ...TaskOpt) *Task {
	build := []TaskOpt{withTaskConfig(src), WithGlobals(g)}
	build = append(build, opts...)
	build = append(build, WithDefaults())
	return NewTask(src.Name, build...)
}

// withTaskConfig seeds a task from the values of an existing task.
func withTaskConfig(src *Task) TaskOpt {
	return func(t *Task) {
		t.Description = src.Description
		t.Source = src.Source
		t.ContinuePolicy = src.ContinuePolicy
		t.Network = src.Network
		t.NoProxy = src.NoProxy
		t.Agent = src.Agent
		t.Resources = src.Resources
		t.Git = src.Git
	}
}

// EnsureSource sets Source.LocalPath to dir when neither LocalPath nor Repo is configured.
func (t *Task) EnsureSource(dir string) {
	if dir != "" && t.Source.LocalPath == "" && t.Source.Repo == "" {
		t.Source.LocalPath = dir
	}
}

// WithFallbackSourceDir applies EnsureSource when the task is built.
func WithFallbackSourceDir(sourceDir string) TaskOpt {
	return func(t *Task) { t.EnsureSource(sourceDir) }
}

// WithModel overrides the agent model. An empty value is ignored. The model string
// is taken as provided; it is not checked against the provider's catalogue.
func WithModel(model string) TaskOpt {
	return func(t *Task) {
		if model == "" {
			return
		}
		if t.Agent == nil {
			t.Agent = &Agent{}
		}
		t.Agent.Provider.Model = model
	}
}

// WithDefaults sets reasonable default values to the task if not configured.
func WithDefaults() TaskOpt {
	return func(t *Task) {
		if t.Agent == nil {
			t.Agent = &Agent{}
		}
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
		if len(t.Agent.Permissions) == 0 {
			t.Agent.Permissions = DefaultPermissions
		}
		if !t.NoProxy && len(t.Network.Proxy.AllowList) == 0 {
			t.Network.Proxy.AllowList = DefaultAllowList
		}
		if t.Resources == nil {
			t.Resources = NewResources()
		} else {
			if t.Resources.CPU == "" {
				t.Resources.CPU = DefaultCPUs
			}
			if t.Resources.Memory == "" {
				t.Resources.Memory = DefaultMemory
			}
		}
	}
}

// Validator is implemented by config sections that can self-check.
type Validator interface {
	Validate() error
}

// Validate runs v.Validate, guarding against an unset section.
func Validate(v Validator) error {
	if v == nil {
		return errors.New("validated config cannot be nil")
	}
	return v.Validate()
}

// Validate checks that the task configuration is ready to start a container.
// By the time it is called, defaults and global settings must already be applied,
// so every nil-able section is expected to be populated.
func (t *Task) Validate() error {
	configs := []Validator{&t.Source, t.Agent, t.Resources, t.Git}
	for _, c := range configs {
		if err := Validate(c); err != nil {
			return err
		}
	}
	return nil
}

// PolicyRequiresNew returns true if the task config requires restart instead of resuming.
func (t *Task) PolicyRequiresNew() bool {
	return t.ContinuePolicy != ContinuePolicyResume
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
	if s == nil {
		return errors.New("source is not configured")
	}
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

// Agent configures the agent that executes the task.
// It includes provider and model selection, turn limits, environment
// variables, and permission settings.
type Agent struct {
	Provider    provider.Provider `yaml:",inline"`
	Image       string            `yaml:"image,omitempty"`
	Language    string            `yaml:"language,omitempty"`
	MaxTurns    int               `yaml:"max_turns,omitempty"`
	TaskPrompt  string            `yaml:"task_prompt,omitempty"`
	EnvVars     map[string]string `yaml:"env_vars,omitempty"`
	Permissions []string          `yaml:"permissions,omitempty"`
}

func (a *Agent) Validate() error {
	if a == nil {
		return errors.New("agent is not configured")
	}
	if a.Provider.Unknown() {
		return errors.New("agent provider is not configured")
	}
	return nil
}

// Resources defines CPU and memory limits for the task container.
type Resources struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// NewResources creates task resources with default values.
func NewResources() *Resources {
	return &Resources{
		CPU:    DefaultCPUs,
		Memory: DefaultMemory,
	}
}

func (r *Resources) Validate() error {
	if r == nil {
		return errors.New("resources are not configured")
	}
	if r.CPU == "" || r.Memory == "" {
		return errors.New("resources cpu and memory are required")
	}
	return nil
}

// GitUser holds git identity configuration for commits made during task execution.
type GitUser struct {
	Name  string `yaml:"user_name,omitempty"`
	Email string `yaml:"user_email,omitempty"`
}

func (user *GitUser) Validate() error {
	if user == nil {
		return errors.New("git user is not configured")
	}
	return nil
}

// Proxy configures the network proxy used to filter task network access.
type Proxy struct {
	Image     string   `yaml:"image,omitempty"`
	Port      int      `yaml:"port,omitempty"`
	AllowList []string `yaml:"base_allowlist,omitempty"`
}

// configured reports whether the proxy has any task-level settings.
func (p Proxy) configured() bool {
	return p.Image != "" || p.Port != 0 || len(p.AllowList) != 0
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
