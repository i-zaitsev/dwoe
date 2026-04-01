// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build integration

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

var (
	testClient *Client
)

func TestMain(m *testing.M) {
	c, err := NewClient()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "docker unavailable: %v\n", err)
		os.Exit(0)
	}
	testClient = c
	defer func() {
		_ = testClient.Close()
		testClient = nil
	}()
	os.Exit(m.Run())
}

func TestClient_Ping(t *testing.T) {
	assert.NotErr(t, testClient.Ping(context.Background()))
}

func TestClient_BuildImage(t *testing.T) {
	dir := t.TempDir()
	dockerfile := filepath.Join(dir, "Dockerfile")
	assert.NotErr(t, os.WriteFile(dockerfile, []byte("FROM alpine\n"), 0644))

	tag := "test-build-" + time.Now().Format("150405")
	assert.NotErr(t, testClient.BuildImage(context.Background(), dockerfile, tag, io.Discard))
}

func TestClient_ContainerLifecycle(t *testing.T) {
	ctx := context.Background()
	name := "test-lifecycle-" + time.Now().Format("150405")
	state, err := Run(
		ctx, testClient,
		CreateContainer(&ContainerConfig{Image: "alpine", Name: name}),
		InspectContainer(),
		StartContainer(),
		ContainerLogs(false),
		StopContainer(5*time.Second),
		WaitContainer(),
		RemoveContainer(true),
	)
	if state != nil && state.ContainerID != "" && err != nil {
		if errRemove := testClient.RemoveContainer(ctx, state.ContainerID, true); errRemove != nil {
			t.Fatal(errRemove)
		}
	}
	assert.NotErr(t, err)
}

func TestClient_NetworkLifecycle(t *testing.T) {
	ctx := context.Background()
	name := "test-net-" + time.Now().Format("150405")
	cfg := &NetworkConfig{
		Name: name,
	}

	state, err := Run(ctx, testClient,
		CreateNetwork(cfg),
		GetNetworkID(name),
		RemoveNetwork(),
	)
	assert.NotErr(t, err)
	if state.NetworkID == "" {
		t.Fatal("expected network ID")
	}
}

func TestClient_NetworkAlreadyExists(t *testing.T) {
	ctx := context.Background()
	name := "test-net-exists-" + time.Now().Format("150405")
	cfg := &NetworkConfig{
		Name: name,
	}

	id1, err := testClient.CreateNetwork(ctx, cfg)
	assert.NotErr(t, err)
	id2, err := testClient.CreateNetwork(ctx, cfg)
	assert.NotErr(t, err)
	assert.Equal(t, id1, id2)
	assert.NotErr(t, testClient.RemoveNetwork(ctx, id1))
}

func TestClient_ContainerNetworkLifecycle(t *testing.T) {
	ctx := context.Background()
	ts := time.Now().Format("150405")
	netCfg := &NetworkConfig{
		Name:    "test-net-" + ts,
		Subnet:  "10.200.0.0/24",
		Gateway: "10.200.0.1",
	}
	containerCfg := &ContainerConfig{
		Image: "alpine",
		Name:  "test-cn-" + ts,
	}

	state, err := Run(ctx, testClient,
		CreateNetwork(netCfg),
		CreateContainer(containerCfg),
		ConnectContainerToNetwork("10.200.0.100"),
		RemoveContainer(true),
		RemoveNetwork(),
	)
	assert.NotErr(t, err)
	if state.NetworkID == "" {
		t.Fatal("expected network ID")
	}
	if state.ContainerID == "" {
		t.Fatal("expected container ID")
	}
}
