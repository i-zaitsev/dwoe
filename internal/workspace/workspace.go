// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package workspace

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/config/provider"
	"github.com/i-zaitsev/dwoe/internal/docker"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/template"
)

// Workspace combines the configured state with task definition.
// Parameters defined in state.Workspace are combined with the allocated task.
// This type is exposed and used by a manager when new workspaces are created from
// the task config.
type Workspace struct {
	*state.Workspace
	Config *config.Task
	Done   bool
}

func New(ws *state.Workspace, cfg *config.Task) *Workspace {
	return &Workspace{Workspace: ws, Config: cfg}
}

// WorkDir returns the path to the workspace's working directory inside the base path.
func (ws *Workspace) WorkDir() string {
	return filepath.Join(ws.BasePath, "workspace")
}

// WorkFile returns a path inside the workspace's working directory.
func (ws *Workspace) WorkFile(parts ...string) string {
	return filepath.Join(ws.WorkDir(), filepath.Join(parts...))
}

// TemplateData builds the template rendering context from workspace state and config.
func (ws *Workspace) TemplateData() *template.Data {
	domains := slices.Concat(ws.Config.Network.Proxy.AllowList, ws.Config.Network.AllowListExtra)
	return &template.Data{
		WorkspaceID:    ws.ID,
		WorkspaceName:  ws.Name,
		Model:          ws.Config.Agent.Provider.Model,
		MaxTurns:       ws.Config.Agent.MaxTurns,
		ProxyIP:        ws.Config.Network.Gateway,
		ProxyPort:      proxyPort(ws.Config.Network.Proxy),
		AllowedDomains: domains,
		GitUserName:    ws.Config.Git.Name,
		GitUserEmail:   ws.Config.Git.Email,
		Env:            ws.Config.Agent.EnvVars,
		Permissions:    ws.Config.Agent.Permissions,
	}
}

type envPair struct {
	key, value string
}

// Env returns the environment variables for the agent container.
func (ws *Workspace) Env() []string {
	p := ws.provider()
	pairs := []envPair{
		{"WORKSPACE_ID", ws.ID},
		{"WORKSPACE_NAME", ws.Name},
		{"AGENT_PROVIDER", p.Name.String()},
		{"AGENT_MODEL", p.Model},
		{"CLAUDE_MODEL", p.Model},
		{"MAX_TURNS", strconv.Itoa(ws.Config.Agent.MaxTurns)},
		{"GIT_USER_NAME", ws.Config.Git.Name},
		{"GIT_USER_EMAIL", ws.Config.Git.Email},
	}
	if prompt := ws.taskPrompt(); prompt != "" {
		pairs = append(pairs, envPair{"TASK_PROMPT", prompt})
	}
	if !ws.Config.NoProxy {
		url := ws.proxyURL()
		pairs = append(pairs, envPair{"HTTP_PROXY", url}, envPair{"HTTPS_PROXY", url})
	}
	pairs = append(pairs, providerAuthEnv(p)...)
	for k, v := range ws.Config.Agent.EnvVars {
		expanded := os.ExpandEnv(v)
		if expanded == "" && strings.Contains(v, "$") {
			slog.Warn("env var resolved to empty", "key", k, "template", v)
		}
		pairs = append(pairs, envPair{k, expanded})
	}
	return formatEnv(pairs)
}

// provider returns the task's resolved provider. The config snapshot is already
// resolved at creation time, so no registry lookup is needed here.
func (ws *Workspace) provider() provider.Provider {
	return ws.Config.Agent.Provider
}

// providerAuthEnv passes the provider's credentials through from the host.
// Each AuthEnvVars entry present in the host environment is forwarded as-is.
func providerAuthEnv(p provider.Provider) []envPair {
	var pairs []envPair
	for _, key := range p.AuthEnvVars {
		if val := os.Getenv(key); val != "" {
			pairs = append(pairs, envPair{key, val})
		}
	}
	return pairs
}

func (ws *Workspace) taskPrompt() string {
	if ws.Config.Agent.TaskPrompt != "" {
		return ws.Config.Agent.TaskPrompt
	}
	if ws.Config.Source.PromptFile != "" {
		return fmt.Sprintf("Follow the instructions in %s", ws.Config.Source.PromptFile)
	}
	slog.Warn("workspace: empty task prompt", "id", ws.ID, "name", ws.Name)
	return ""
}

func (ws *Workspace) proxyURL() string {
	host := ws.Config.Network.Gateway
	if host == "" {
		host = proxyContainerName(ws.Name)
	}
	return fmt.Sprintf("http://%s:%d", host, proxyPort(ws.Config.Network.Proxy))
}

func proxyPort(p config.Proxy) int {
	if p.Port != 0 {
		return p.Port
	}
	return config.DefaultProxyPort
}

func formatEnv(pairs []envPair) []string {
	out := make([]string, len(pairs))
	for i, p := range pairs {
		out[i] = fmt.Sprintf("%s=%s", p.key, p.value)
	}
	return out
}

// Mounts returns the volume mounts for the agent container.
// The settings.json mount is Claude-specific; other providers do not use it.
func (ws *Workspace) Mounts() []docker.Mount {
	mounts := []docker.Mount{
		{Source: filepath.Join(ws.BasePath, "workspace"), Target: "/workspace"},
		{Source: filepath.Join(ws.BasePath, "logs", "agent"), Target: "/logs"},
	}
	if ws.provider().Name == provider.ModelProviderAnthropic {
		mounts = append(mounts, docker.Mount{
			Source:   filepath.Join(ws.BasePath, "settings.json"),
			Target:   "/home/agent/.claude/settings.json",
			ReadOnly: true,
		})
	}
	return mounts
}
