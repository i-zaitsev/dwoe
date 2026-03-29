// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package workspace orchestrates the lifecycle of isolated Docker-based workspaces.
package workspace

import (
	"context"
	"io"
	"time"

	"github.com/i-zaitsev/dwoe/internal/docker"
	"github.com/i-zaitsev/dwoe/internal/state"
)

// DockerClient defines interaction with Docker API.
//
// Manager expects a real Docker client. The interface decouples it from the real type
// for testing purposes. See internal/docker/docker.Client which is a light-weight proxy
// on top of the official Docker SDK.
type DockerClient interface {
	Ping(ctx context.Context) error
	CreateContainer(ctx context.Context, cfg *docker.ContainerConfig) (string, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string, timeout time.Duration) error
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	CreateNetwork(ctx context.Context, cfg *docker.NetworkConfig) (string, error)
	RemoveNetwork(ctx context.Context, networkID string) error
	ContainerLogs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error)
	WaitContainer(ctx context.Context, containerID string) (int, error)
	InspectContainer(ctx context.Context, containerID string) (docker.ContainerInfo, error)
	Close() error
}

// StateStore defines state persistence operations.
// See internal/state/state.Store implementing file-based state persistence.
type StateStore interface {
	Save(ws *state.Workspace) error
	Load(id string) (*state.Workspace, error)
	List() ([]*state.Workspace, error)
	Delete(id string) error
	UpdateStatus(id, status string) error
}
