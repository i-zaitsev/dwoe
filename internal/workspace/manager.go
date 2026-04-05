// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/google/uuid"
	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/docker"
	"github.com/i-zaitsev/dwoe/internal/namegen"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/template"
	"gopkg.in/yaml.v3"
)

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusStopped   = "stopped"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// Manager orchestrates workspace lifecycle: creation, starting, stopping,
// and cleanup of isolated Docker-based workspaces.
type Manager struct {
	dataDir string
	cli     DockerClient
	state   StateStore
}

// NewManager creates a Manager backed by a real Docker client and file-based state store.
// The dataDir is used for workspace directories and state persistence.
func NewManager(dataDir string) (*Manager, error) {
	cli, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("cannot create manager: %w", err)
	}
	if err := cli.Ping(context.Background()); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("cannot connect to Docker: %w", err)
	}
	store := state.NewStore(dataDir)
	m, err := newManager(dataDir, cli, store)
	if err != nil {
		_ = cli.Close()
		return nil, err
	}
	return m, nil
}

// NewManagerWith creates a Manager with the provided Docker client and state store.
func NewManagerWith(dataDir string, cli DockerClient, state StateStore) (*Manager, error) {
	return newManager(dataDir, cli, state)
}

// Create creates a new workspace from the given task configuration.
// It generates a unique name, creates the directory structure, copies source files,
// and persists both the config and initial state. On any error, the workspace
// directory is cleaned up automatically.
func (m *Manager) Create(cfg *config.Task) (*Workspace, error) {
	slog.Info("workspace: create", "name", cfg.Name)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("workspace: task config: %w", err)
	}

	cfg.Name = m.uniqueName(cfg.Name)

	id := uuid.New().String()
	basePath := filepath.Join(m.dataDir, "workspaces", id)

	success := false
	defer func() {
		if !success {
			if errRemove := os.RemoveAll(basePath); errRemove != nil {
				slog.Error("workspace: remove dir", "err", errRemove)
			}
		}
	}()

	if err := initDirs(basePath); err != nil {
		return nil, err
	}
	if err := populateSource(cfg, filepath.Join(basePath, "workspace")); err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}

	if err := storeConfig(basePath, cfg); err != nil {
		return nil, fmt.Errorf("workspace: store config: %w", err)
	}

	now := time.Now()
	ws := &state.Workspace{
		ID:        id,
		Name:      cfg.Name,
		Status:    StatusPending,
		BasePath:  basePath,
		CreatedAt: &now,
	}
	if err := m.state.Save(ws); err != nil {
		return nil, fmt.Errorf("workspace: save state: %w", err)
	}

	success = true // otherwise, deletes the base path to clean up
	return &Workspace{
		Workspace: ws,
		Config:    cfg,
	}, nil
}

// uniqueName returns a workspace name that doesn't collide with existing ones.
// If the requested name is empty or already taken, a random name is generated.
func (m *Manager) uniqueName(requested string) string {
	if requested == "" {
		// no name provided: generate a random one, with safety check
		// to avoid (unlikely) collisions
		uniq := namegen.Generate()
		existing, _ := m.state.List()
		for {
			taken := false
			for _, ws := range existing {
				if ws.Name == requested {
					taken = true
					break
				}
			}
			if !taken {
				return uniq
			}
			uniq = namegen.Generate()
		}
	}

	// the name coming from task file: do not generate but check
	// if already taken, e.g., the task was executed before; append
	// suffix to avoid name collision or naming into a random string
	existing, _ := m.state.List()
	var same []string
	for _, ws := range existing {
		if strings.HasPrefix(ws.Name, requested) {
			same = append(same, ws.Name)
		}
	}
	if len(same) > 0 {
		return fmt.Sprintf("%s-%d", requested, len(same))
	}
	return requested
}

// initDirs creates the standard subdirectory layout under basePath:
// workspace/, logs/agent/, logs/proxy/, and .docker/.
func initDirs(basePath string) error {
	for _, sub := range []string{
		"workspace",
		filepath.Join("logs", "agent"),
		filepath.Join("logs", "proxy"),
		".docker",
	} {
		if err := os.MkdirAll(filepath.Join(basePath, sub), 0o755); err != nil {
			return fmt.Errorf("workspace: subdir %s: %w", sub, err)
		}
	}
	return nil
}

// populateSource copies or clones the task's source code into workspaceDir,
// then copies any prompt and spec files alongside it.
func populateSource(cfg *config.Task, workspaceDir string) error {
	if cfg.Source.LocalPath != "" {
		if err := CopyLocalDir(cfg.Source.LocalPath, workspaceDir); err != nil {
			return fmt.Errorf("copy source: %w", err)
		}
	} else if cfg.Source.Repo != "" {
		if err := CloneRepo(cfg.Source.Repo, workspaceDir, cfg.Source.Branch); err != nil {
			return fmt.Errorf("clone source: %w", err)
		}
	}
	for _, src := range []string{cfg.Source.PromptFile, cfg.Source.SpecFile} {
		if src == "" {
			continue
		}
		dst := filepath.Join(workspaceDir, filepath.Base(src))
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read %s: %w", src, err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
	}
	return nil
}

// Get loads a workspace by ID, reading both its persisted state and config.
// Defaults are applied to the config before returning.
func (m *Manager) Get(id string) (*Workspace, error) {
	slog.Debug("workspace: get", "id", id)
	ws, err := m.state.Load(id)
	if err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}
	cfgTask, err := loadConfig(ws.BasePath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	cfgTask.ApplyDefaults()
	return &Workspace{
		Workspace: ws,
		Config:    cfgTask,
	}, nil
}

// GetByName finds a workspace by its human-readable name.
// Returns a NotFoundError if no workspace matches.
func (m *Manager) GetByName(name string) (*Workspace, error) {
	states, err := m.state.List()
	if err != nil {
		return nil, fmt.Errorf("state: %w", err)
	}
	for _, ws := range states {
		if ws.Name == name {
			return m.Get(ws.ID)
		}
	}
	return nil, &state.NotFoundError{ID: name}
}

// Resolve looks up a workspace by exact ID, then exact name, then ID prefix.
func (m *Manager) Resolve(nameOrID string) (*Workspace, error) {
	ws, err := m.Get(nameOrID)
	if err == nil {
		return ws, nil
	}
	ws, err = m.GetByName(nameOrID)
	if err == nil {
		return ws, nil
	}
	return m.ResolvePrefix(nameOrID)
}

// ResolvePrefix finds a workspace whose ID starts with the given prefix.
// Returns AmbiguousMatchError when multiple workspaces match.
func (m *Manager) ResolvePrefix(prefix string) (*Workspace, error) {
	states, err := m.state.List()
	if err != nil {
		return nil, fmt.Errorf("state: %w", err)
	}
	var matches []*state.Workspace
	for _, ws := range states {
		if strings.HasPrefix(ws.ID, prefix) {
			matches = append(matches, ws)
		}
	}
	switch len(matches) {
	case 0:
		return nil, &state.NotFoundError{ID: prefix}
	case 1:
		return m.Get(matches[0].ID)
	default:
		ids := make([]string, len(matches))
		for i, ws := range matches {
			ids[i] = ws.ID
		}
		return nil, &state.AmbiguousMatchError{Prefix: prefix, IDs: ids}
	}
}

// ResolveCompleted resolves a workspace and verifies it is no longer running or pending.
// Returns an error if the workspace is still active.
func (m *Manager) ResolveCompleted(nameOrID string) (*Workspace, error) {
	ws, err := m.Resolve(nameOrID)
	if err != nil {
		return nil, err
	}
	if ws.Status == StatusRunning || ws.Status == StatusPending {
		return nil, fmt.Errorf("workspace %s is still %s", ws.Name, ws.Status)
	}
	return ws, nil
}

// List returns all known workspaces with their configs loaded.
func (m *Manager) List() ([]*Workspace, error) {
	states, err := m.state.List()
	if err != nil {
		return nil, fmt.Errorf("state: %w", err)
	}
	out := make([]*Workspace, 0, len(states))
	for _, ws := range states {
		cfgTask, errLoad := loadConfig(ws.BasePath)
		if errLoad != nil {
			return nil, fmt.Errorf("load config: %w", errLoad)
		}
		out = append(out, &Workspace{Workspace: ws, Config: cfgTask})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].CreatedAt, out[j].CreatedAt
		if a == nil && b == nil {
			return out[i].Name < out[j].Name
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		if !a.Equal(*b) {
			return a.After(*b)
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Start launches a workspace: creates a Docker network, renders config templates,
// starts proxy and agent containers, and saves the running state.
// The workspace must be in pending or stopped status.
// On failure, all created resources are cleaned up.
func (m *Manager) Start(ctx context.Context, id string) error {
	slog.Info("workspace: start", "id", id)
	ws, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if ws.Status != StatusPending && ws.Status != StatusStopped {
		return fmt.Errorf("start: invalid status %q", ws.Status)
	}

	netID, err := m.setupNetwork(ctx, ws)
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// ctxNoCancel is used for cleanup calls so that a cancelled ctx (e.g. SIGINT
	// arriving mid-startup) does not prevent Docker resources from being removed.
	ctxNoCancel := context.WithoutCancel(ctx)
	success := false
	defer func() {
		if success {
			return
		}
		if err := m.cli.RemoveNetwork(ctxNoCancel, netID); err != nil {
			slog.Warn("start: cleanup network", "err", err)
		}
	}()

	if err := template.WriteAll(ws.BasePath, ws.TemplateData()); err != nil {
		return fmt.Errorf("start: render templates: %w", err)
	}
	claudeSrc := filepath.Join(ws.BasePath, "CLAUDE.md")
	claudeDst := filepath.Join(ws.BasePath, "workspace", "CLAUDE.md")
	if data, errRead := os.ReadFile(claudeSrc); errRead == nil {
		if errWrite := os.WriteFile(claudeDst, data, 0o644); errWrite != nil {
			slog.Warn("start: write CLAUDE.md to workspace", "err", errWrite)
		}
	}

	containerIDs, err := m.startContainers(ctx, ws, netID)
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	now := time.Now()
	ws.Status = StatusRunning
	ws.StartedAt = &now
	ws.ContainerIDs = containerIDs
	ws.NetworkID = netID

	if errSave := m.saveState(ws); errSave != nil {
		return fmt.Errorf("start: save workspace state: %w", errSave)
	}

	success = true
	return nil
}

// Stop gracefully stops a running workspace's containers within the given timeout.
// The agent container is stopped first, followed by the proxy if present.
// Stop is a no-op if the workspace is already stopped.
func (m *Manager) Stop(ctx context.Context, id string, timeout time.Duration) error {
	slog.Info("workspace: stop", "id", id)
	ws, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("stop: %w", err)
	}
	if ws.Status == StatusStopped {
		return nil
	}
	if ws.Status != StatusRunning {
		return fmt.Errorf("stop: invalid status %q", ws.Status)
	}
	if ws.ContainerIDs == nil {
		return fmt.Errorf("stop: container IDs not set")
	}

	agentID, ok := ws.ContainerIDs["agent"]
	if !ok || agentID == "" {
		return fmt.Errorf("stop: agent container ID not found")
	}
	if err := m.cli.StopContainer(ctx, agentID, timeout); err != nil {
		return fmt.Errorf("stop: agent: %w", err)
	}
	if proxyID, ok := ws.ContainerIDs["proxy"]; ok && proxyID != "" {
		if err := m.cli.StopContainer(ctx, proxyID, 5*time.Second); err != nil {
			slog.Warn("stop: stop proxy", "err", err)
		}
	}

	now := time.Now()
	ws.Status = StatusStopped
	ws.FinishedAt = &now
	ws.ErrorMsg = "stopped by user"
	if errSave := m.saveState(ws); errSave != nil {
		return fmt.Errorf("stop: save workspace state: %w", errSave)
	}
	return nil
}

// DestroyOpts configures Destroy.
type DestroyOpts struct {
	KeepDir bool
}

// Destroy removes a workspace entirely: containers, network, state, and optionally
// the workspace directory on disk. Collects all errors and returns them joined.
func (m *Manager) Destroy(ctx context.Context, id string, opts DestroyOpts) error {
	slog.Info("workspace: destroy", "id", id, "keep_dir", opts.KeepDir)

	ws, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("workspace: destroy: %w", err)
	}

	errs := m.removeDockerResources(ctx, ws.Workspace)
	errs = append(errs, m.state.Delete(ws.ID))

	if !opts.KeepDir {
		errs = append(errs, os.RemoveAll(ws.BasePath))
	}

	if errAll := errors.Join(errs...); errAll != nil {
		return fmt.Errorf("workspace: destroy: %w", errAll)
	}

	return nil
}

// Cleanup removes Docker resources (containers and network) but preserves the
// workspace directory and state entry. Used after a workspace finishes to free
// Docker resources while keeping results accessible.
func (m *Manager) Cleanup(ctx context.Context, id string) error {
	slog.Info("workspace: cleanup", "id", id)

	ws, errGet := m.Get(id)
	if errGet != nil {
		return fmt.Errorf("workspace: cleanup: %w", errGet)
	}

	m.saveContainerLogs(ctx, ws.Workspace)

	errs := m.removeDockerResources(ctx, ws.Workspace)

	ws.ContainerIDs = nil
	ws.NetworkID = ""

	errs = append(errs, m.state.Save(ws.Workspace))

	if errAll := errors.Join(errs...); errAll != nil {
		return fmt.Errorf("workspace: cleanup: %w", errAll)
	}

	return nil
}

// Logs returns a reader for the agent container's stdout/stderr.
// When follow is true, the reader streams logs as they are written.
func (m *Manager) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	slog.Info("workspace: logs", "id", id, "follow", follow)

	agentID, err := m.getAgentID(id)
	if err != nil {
		return m.savedLogs(id)
	}

	return m.cli.ContainerLogs(ctx, agentID, follow)
}

// Wait blocks until the agent container exits and returns its exit code.
// The workspace status is updated to completed (exit 0) or failed (non-zero).
func (m *Manager) Wait(ctx context.Context, id string) (int, error) {
	slog.Info("workspace: wait", "id", id)

	agentID, err := m.getAgentID(id)
	if err != nil {
		return 0, fmt.Errorf("wait: agent ID: %w", err)
	}

	// exit code of the container from the docker sdk
	code, err := m.cli.WaitContainer(ctx, agentID)
	if err != nil {
		slog.Error("wait: container code", "exitCode", code)
		return 0, fmt.Errorf("wait: container: %w", err)
	}

	ws, errLoad := m.state.Load(id)
	if errLoad != nil {
		return code, nil
	}

	ws.ExitCode = &code
	if code != 0 {
		ws.ErrorMsg = fmt.Sprintf("exit code %d", code)
	}

	setFinishedWithCode(ws, code)

	if errSave := m.state.Save(ws); errSave != nil {
		slog.Error("wait: save state", "err", errSave)
	}

	return code, nil
}

// Sync updates the workspace information.
// The source of truth about the worker is the state of the docker container.
// Ensures that containers in StatusRunning are actually still working.
func (m *Manager) Sync(ctx context.Context, id string) error {
	slog.Info("workspace: sync", "id", id)

	ws, err := m.state.Load(id)
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	if ws.Status != StatusRunning {
		return nil
	}

	agentID, ok := ws.ContainerIDs["agent"]
	if !ok || agentID == "" {
		return nil
	}

	info, err := m.cli.InspectContainer(ctx, agentID)
	if err != nil {
		slog.Warn("sync: inspect failed, marking as failed", "id", id, "err", err)
		m.saveContainerLogs(ctx, ws)
		m.removeDockerResources(ctx, ws)
		now := time.Now()
		ws.Status = StatusFailed
		ws.FinishedAt = &now
		ws.ContainerIDs = nil
		ws.NetworkID = ""
		return m.state.Save(ws)
	}

	if info.Status == StatusRunning {
		return nil
	}

	setFinishedWithCode(ws, info.ExitCode)
	m.saveContainerLogs(ctx, ws)
	m.removeDockerResources(ctx, ws)
	ws.ContainerIDs = nil
	ws.NetworkID = ""
	return m.state.Save(ws)
}

func setFinishedWithCode(ws *state.Workspace, code int) {
	slog.Debug("set workspace state", "exitCode", code, "ok", code == 0)
	now := time.Now()
	ws.FinishedAt = &now
	if code == 0 {
		ws.Status = StatusCompleted
	} else {
		ws.Status = StatusFailed
	}
}

// SyncAll synchronizes the state of all running workspaces with Docker.
func (m *Manager) SyncAll(ctx context.Context) error {
	slog.Info("workspace: sync-all")

	states, err := m.state.List()
	if err != nil {
		return fmt.Errorf("sync-all: %w", err)
	}

	for _, ws := range states {
		if ws.Status != StatusRunning {
			continue
		}
		if errSync := m.Sync(ctx, ws.ID); errSync != nil {
			slog.Warn("workspace: sync-all", "id", ws.ID, "err", errSync)
		}
	}

	return nil
}

func (m *Manager) BatchRecords() ([]*batch.Record, error) {
	return batch.ReadRecords(m.dataDir, "")
}

// Diff reads changes made in the workspace for inspection.
func (m *Manager) Diff(id string) (*DiffInfo, error) {
	ws, err := m.state.Load(id)
	if err != nil {
		return nil, fmt.Errorf("diff: %w", err)
	}
	repoDir := filepath.Join(ws.BasePath, "workspace")
	root, err := gitOutput("-C", repoDir, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("diff: root: %w", err)
	}
	logOut, err := gitFullOutput("-C", repoDir, "log", "--oneline", root+"..HEAD")
	if err != nil {
		return nil, fmt.Errorf("diff: log: %w", err)
	}
	stat, err := gitFullOutput("-C", repoDir, "diff", "--stat", root, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("diff: stat: %w", err)
	}
	diff, err := gitFullOutput("-C", repoDir, "diff", root, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("diff: %w", err)
	}
	return &DiffInfo{
		Commits: parseCommits(logOut),
		Stat:    stat,
		Diff:    diff,
	}, nil
}

func parseCommits(log string) []CommitInfo {
	if log == "" {
		return nil
	}
	var commits []CommitInfo
	for _, line := range strings.Split(log, "\n") {
		hash, msg, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		commits = append(commits, CommitInfo{Hash: hash, Message: msg})
	}
	return commits
}

// UpdateStatus sets the workspace status in the state store.
func (m *Manager) UpdateStatus(id, status string) error {
	return m.state.UpdateStatus(id, status)
}

// saveState persists the workspace's current state to the store.
func (m *Manager) saveState(ws *Workspace) error {
	return m.state.Save(ws.Workspace)
}

// removeDockerResources force-removes all containers and the network associated
// with a workspace. Returns any errors encountered during removal.
func (m *Manager) removeDockerResources(ctx context.Context, ws *state.Workspace) []error {
	var errs []error
	for _, cid := range ws.ContainerIDs {
		if cid == "" {
			continue
		}
		if err := m.cli.RemoveContainer(ctx, cid, true); err != nil {
			errs = append(errs, err)
		}
	}
	if ws.NetworkID == "" {
		return errs
	}
	if err := m.cli.RemoveNetwork(ctx, ws.NetworkID); err != nil {
		errs = append(errs, err)
	}
	return errs
}

// startProxy creates and starts the squid proxy container on the given network.
// Falls back to default image and port when not configured.
func (m *Manager) startProxy(ctx context.Context, ws *Workspace, netID string) (string, error) {
	proxy := ws.Config.Network.Proxy
	proxyImage := proxy.Image
	if proxyImage == "" {
		proxyImage = config.DefaultProxyImage
	}
	port := proxyPort(proxy)

	proxyMounts := []docker.Mount{
		{Source: filepath.Join(ws.BasePath, "proxy", "squid.conf"), Target: "/etc/squid/squid.conf", ReadOnly: true},
		{Source: filepath.Join(ws.BasePath, "proxy", "allowlist.txt"), Target: "/etc/squid/allowlist.txt", ReadOnly: true},
		{Source: filepath.Join(ws.BasePath, "logs", "proxy"), Target: "/var/log/squid"},
	}

	proxyID, err := m.cli.CreateContainer(ctx, &docker.ContainerConfig{
		Image:     proxyImage,
		Name:      proxyContainerName(ws.Name),
		Mounts:    proxyMounts,
		NetworkID: netID,
	})
	if err != nil {
		return "", fmt.Errorf("proxy create: %w", err)
	}
	if err := m.cli.StartContainer(ctx, proxyID); err != nil {
		if errClean := m.cli.RemoveContainer(ctx, proxyID, true); errClean != nil {
			slog.Warn("start: cleanup proxy", "err", errClean)
		}
		return "", fmt.Errorf("proxy: %w", err)
	}

	slog.Info("workspace: proxy started", "id", proxyID, "image", proxyImage, "port", port)
	return proxyID, nil
}

// setupNetwork creates a Docker network for the workspace.
// Uses the configured network name or falls back to `{id}_default`.
func (m *Manager) setupNetwork(ctx context.Context, ws *Workspace) (string, error) {
	cfgNet := ws.Config.Network
	if cfgNet.Name == "" {
		cfgNet.Name = fmt.Sprintf("%s_default", ws.ID)
	}
	return m.cli.CreateNetwork(ctx, &docker.NetworkConfig{
		Name:    cfgNet.Name,
		Subnet:  cfgNet.Subnet,
		Gateway: cfgNet.Gateway,
	})
}

// startContainers launches the proxy (unless disabled) and agent containers on netID.
// Returns a map of the role ("agent", "proxy") to container ID.
// On failure, any already-created containers are cleaned up.
func (m *Manager) startContainers(ctx context.Context, ws *Workspace, netID string) (map[string]string, error) {

	// ctxNoCancel is used in cleanup so that a canceled ctx
	// (e.g., SIGINT arriving mid-startup) does not prevent
	// already-created containers from being removed.
	ctxNoCancel := context.WithoutCancel(ctx)

	containerIDs := map[string]string{}

	cleanup := func() {
		// cleans up containers with creation failed
		slog.Warn("start: failed to create container; cleaning up", "netID", netID)
		for _, cid := range containerIDs {
			if err := m.cli.RemoveContainer(ctxNoCancel, cid, true); err != nil {
				slog.Warn("start: failed to clean up container", "id", cid, "err", err)
			}
		}
	}

	if !ws.Config.NoProxy {
		slog.Debug("start: proxy enabled", "netID", netID)
		proxyID, err := m.startProxy(ctx, ws, netID)
		if err != nil {
			slog.Error("start: proxy", "err", err)
			return nil, err
		}
		containerIDs["proxy"] = proxyID
	}

	conID, errCreate := m.cli.CreateContainer(ctx, &docker.ContainerConfig{
		Image:      ws.Config.Agent.Image,
		Name:       fmt.Sprintf("dwoe-%s-agent", ws.Name),
		Env:        ws.Env(),
		Mounts:     ws.Mounts(),
		NetworkID:  netID,
		Resources:  parseResources(ws.Config.Resources),
		WorkingDir: "/workspace",
	})

	if errCreate != nil {
		cleanup()
		return nil, fmt.Errorf("create container: %w", errCreate)
	}

	containerIDs["agent"] = conID

	if errStart := m.cli.StartContainer(ctx, conID); errStart != nil {
		cleanup()
		return nil, fmt.Errorf("start container: %w", errStart)
	}

	return containerIDs, nil
}

const savedLogFile = "container.log"

func (m *Manager) saveContainerLogs(ctx context.Context, ws *state.Workspace) {
	agentID, ok := ws.ContainerIDs["agent"]
	if !ok || agentID == "" {
		return
	}
	rc, err := m.cli.ContainerLogs(ctx, agentID, false)
	if err != nil {
		slog.Warn("cleanup: save logs", "id", ws.ID, "err", err)
		return
	}
	defer rc.Close()

	logPath := filepath.Join(ws.BasePath, "logs", "agent", savedLogFile)
	f, err := os.Create(logPath)
	if err != nil {
		slog.Warn("cleanup: create log file", "path", logPath, "err", err)
		return
	}
	defer f.Close()
	if _, err := io.Copy(f, rc); err != nil {
		slog.Warn("cleanup: write logs", "path", logPath, "err", err)
	}
}

func (m *Manager) savedLogs(id string) (io.ReadCloser, error) {
	ws, err := m.state.Load(id)
	if err != nil {
		return nil, fmt.Errorf("workspace: logs: %w", err)
	}
	logPath := filepath.Join(ws.BasePath, "logs", "agent", savedLogFile)
	f, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("workspace: logs: %w", err)
	}
	return f, nil
}

// proxyContainerName returns the Docker container name for a workspace's proxy.
func proxyContainerName(wsName string) string {
	return fmt.Sprintf("dwoe-%s-proxy", wsName)
}

// getAgentID resolves the agent container ID for the given workspace.
func (m *Manager) getAgentID(workspaceID string) (string, error) {
	ws, err := m.Get(workspaceID)
	if err != nil {
		return "", err
	}
	agentID, ok := ws.ContainerIDs["agent"]
	if !ok || agentID == "" {
		return "", fmt.Errorf("agent container not found for workspace ID %s", workspaceID)
	}
	return agentID, nil
}

// newManager is the internal constructor shared by NewManager and NewManagerWith.
func newManager(dataDir string, cli DockerClient, state StateStore) (*Manager, error) {
	return &Manager{
		dataDir: dataDir,
		cli:     cli,
		state:   state,
	}, nil
}

// parseResources converts string-based CPU and memory limits from the task config
// into the numeric types expected by the Docker API. Invalid values are silently ignored.
func parseResources(cfg config.Resources) docker.Resources {
	var res docker.Resources
	if cfg.CPU != "" {
		cpus, err := strconv.ParseFloat(cfg.CPU, 64)
		if err != nil {
			slog.Warn("parseResources: invalid cpu value", "cpu", cfg.CPU, "err", err)
		} else {
			res.CPUs = cpus
		}
	}
	if cfg.Memory != "" {
		mem, err := units.RAMInBytes(cfg.Memory)
		if err != nil {
			slog.Warn("parseResources: invalid memory value", "memory", cfg.Memory, "err", err)
		} else {
			res.Memory = mem
		}
	}
	return res
}

// storeConfig writes the task config as YAML to the workspace's config.yaml.
func storeConfig(basePath string, cfg *config.Task) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(basePath, "config.yaml"), data, 0o644)
}

// loadConfig reads and unmarshals the task config from the workspace's config.yaml.
func loadConfig(basePath string) (*config.Task, error) {
	data, err := os.ReadFile(filepath.Join(basePath, "config.yaml"))
	if err != nil {
		return nil, err
	}
	var cfg config.Task
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
