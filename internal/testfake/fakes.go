// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package testfake provides in-memory fakes for [workspace.DockerClient] and [workspace.StateStore].
package testfake

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/i-zaitsev/dwoe/internal/docker"
	"github.com/i-zaitsev/dwoe/internal/state"
)

// FakeDocker is an in-memory implementation of [workspace.DockerClient] for testing.
type FakeDocker struct {
	Calls              []string
	PingFn             func(ctx context.Context) error
	CreateContainerFn  func(ctx context.Context, cfg *docker.ContainerConfig) (string, error)
	StartContainerFn   func(ctx context.Context, id string) error
	StopContainerFn    func(ctx context.Context, id string, timeout time.Duration) error
	RemoveContainerFn  func(ctx context.Context, id string, force bool) error
	CreateNetworkFn    func(ctx context.Context, cfg *docker.NetworkConfig) (string, error)
	RemoveNetworkFn    func(ctx context.Context, id string) error
	ContainerLogsFn    func(ctx context.Context, id string, follow bool) (io.ReadCloser, error)
	WaitContainerFn    func(ctx context.Context, id string) (int, error)
	InspectContainerFn func(ctx context.Context, id string) (docker.ContainerInfo, error)
}

func (f *FakeDocker) Ping(ctx context.Context) error {
	f.Calls = append(f.Calls, "Ping")
	if f.PingFn != nil {
		return f.PingFn(ctx)
	}
	return nil
}

func (f *FakeDocker) CreateContainer(ctx context.Context, cfg *docker.ContainerConfig) (string, error) {
	f.Calls = append(f.Calls, "CreateContainer")
	if f.CreateContainerFn != nil {
		return f.CreateContainerFn(ctx, cfg)
	}
	return "fake-container", nil
}

func (f *FakeDocker) StartContainer(ctx context.Context, id string) error {
	f.Calls = append(f.Calls, "StartContainer")
	if f.StartContainerFn != nil {
		return f.StartContainerFn(ctx, id)
	}
	return nil
}

func (f *FakeDocker) StopContainer(ctx context.Context, id string, timeout time.Duration) error {
	f.Calls = append(f.Calls, "StopContainer")
	if f.StopContainerFn != nil {
		return f.StopContainerFn(ctx, id, timeout)
	}
	return nil
}

func (f *FakeDocker) RemoveContainer(ctx context.Context, id string, force bool) error {
	f.Calls = append(f.Calls, "RemoveContainer")
	if f.RemoveContainerFn != nil {
		return f.RemoveContainerFn(ctx, id, force)
	}
	return nil
}

func (f *FakeDocker) CreateNetwork(ctx context.Context, cfg *docker.NetworkConfig) (string, error) {
	f.Calls = append(f.Calls, "CreateNetwork")
	if f.CreateNetworkFn != nil {
		return f.CreateNetworkFn(ctx, cfg)
	}
	return "fake-net", nil
}

func (f *FakeDocker) RemoveNetwork(ctx context.Context, id string) error {
	f.Calls = append(f.Calls, "RemoveNetwork")
	if f.RemoveNetworkFn != nil {
		return f.RemoveNetworkFn(ctx, id)
	}
	return nil
}

func (f *FakeDocker) ContainerLogs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	f.Calls = append(f.Calls, "ContainerLogs")
	if f.ContainerLogsFn != nil {
		return f.ContainerLogsFn(ctx, id, follow)
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *FakeDocker) WaitContainer(ctx context.Context, id string) (int, error) {
	f.Calls = append(f.Calls, "WaitContainer")
	if f.WaitContainerFn != nil {
		return f.WaitContainerFn(ctx, id)
	}
	return 0, nil
}

func (f *FakeDocker) InspectContainer(ctx context.Context, id string) (docker.ContainerInfo, error) {
	f.Calls = append(f.Calls, "InspectContainer")
	if f.InspectContainerFn != nil {
		return f.InspectContainerFn(ctx, id)
	}
	return docker.ContainerInfo{Status: "running"}, nil
}

func (f *FakeDocker) Close() error { return nil }

// FakeState is an in-memory implementation of [workspace.StateStore] for testing.
type FakeState struct {
	Data map[string]*state.Workspace
}

// NewFakeState creates an empty FakeState.
func NewFakeState() *FakeState {
	return &FakeState{Data: make(map[string]*state.Workspace)}
}

func (f *FakeState) Save(ws *state.Workspace) error {
	f.Data[ws.ID] = ws
	return nil
}

func (f *FakeState) Load(id string) (*state.Workspace, error) {
	ws, ok := f.Data[id]
	if !ok {
		return nil, &state.NotFoundError{ID: id}
	}
	return ws, nil
}

func (f *FakeState) List() ([]*state.Workspace, error) {
	out := make([]*state.Workspace, 0, len(f.Data))
	for _, ws := range f.Data {
		out = append(out, ws)
	}
	slices.SortFunc(out, func(a, b *state.Workspace) int {
		if a.CreatedAt == nil {
			return 1
		}
		if b.CreatedAt == nil {
			return -1
		}
		ta, tb := *a.CreatedAt, *b.CreatedAt
		if ta.Before(tb) {
			return 1
		} else if ta.Equal(tb) {
			return strings.Compare(a.Name, b.Name)
		}
		return -1
	})
	return out, nil
}

func (f *FakeState) Delete(id string) error {
	delete(f.Data, id)
	return nil
}

func (f *FakeState) UpdateStatus(id, status string) error {
	ws, ok := f.Data[id]
	if !ok {
		return &state.NotFoundError{ID: id}
	}
	ws.Status = status
	return nil
}

// CreateWorkspace creates a minimal workspace directory and returns its state.
func CreateWorkspace(t *testing.T, dir, id, name, status string) *state.Workspace {
	t.Helper()
	wsDir := filepath.Join(dir, id)
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srcDir := t.TempDir()
	cfg := fmt.Sprintf("name: %s\nsource:\n  local_path: %s\n", name, srcDir)
	if err := os.WriteFile(filepath.Join(wsDir, "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	ws := &state.Workspace{
		ID:       id,
		Name:     name,
		Status:   status,
		BasePath: wsDir,
	}
	if status == "running" {
		ws.ContainerIDs = map[string]string{"agent": "fake-container"}
	}
	return ws
}
