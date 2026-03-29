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
}

func New(ws *state.Workspace, cfg *config.Task) *Workspace {
	return &Workspace{Workspace: ws, Config: cfg}
}

// WorkDir returns the path to the workspace's working directory inside the base path.
func (ws *Workspace) WorkDir() string {
	return filepath.Join(ws.BasePath, "workspace")
}

// TemplateData builds the template rendering context from workspace state and config.
func (ws *Workspace) TemplateData() *template.Data {
	domains := slices.Concat(ws.Config.Network.Proxy.AllowList, ws.Config.Network.AllowListExtra)
	return &template.Data{
		WorkspaceID:    ws.ID,
		WorkspaceName:  ws.Name,
		Model:          ws.Config.Agent.Model,
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
	pairs := []envPair{
		{"WORKSPACE_ID", ws.ID},
		{"WORKSPACE_NAME", ws.Name},
		{"CLAUDE_MODEL", ws.Config.Agent.Model},
		{"MAX_TURNS", strconv.Itoa(ws.Config.Agent.MaxTurns)},
		{"GIT_USER_NAME", ws.Config.Git.Name},
		{"GIT_USER_EMAIL", ws.Config.Git.Email},
	}
	if ws.Config.Agent.TaskPrompt != "" {
		pairs = append(pairs, envPair{"TASK_PROMPT", ws.Config.Agent.TaskPrompt})
	}
	if !ws.Config.NoProxy {
		url := ws.proxyURL()
		pairs = append(pairs, envPair{"HTTP_PROXY", url}, envPair{"HTTPS_PROXY", url})
	}
	for k, v := range ws.Config.Agent.EnvVars {
		expanded := os.ExpandEnv(v)
		if expanded == "" && strings.Contains(v, "$") {
			slog.Warn("env var resolved to empty", "key", k, "template", v)
		}
		pairs = append(pairs, envPair{k, expanded})
	}
	return formatEnv(pairs)
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
func (ws *Workspace) Mounts() []docker.Mount {
	return []docker.Mount{
		{Source: filepath.Join(ws.BasePath, "workspace"), Target: "/workspace"},
		{Source: filepath.Join(ws.BasePath, "logs", "agent"), Target: "/logs"},
		{Source: filepath.Join(ws.BasePath, "settings.json"), Target: "/home/agent/.claude/settings.json", ReadOnly: true},
	}
}
