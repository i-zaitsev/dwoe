// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/testfake"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

type testSetup struct {
	docker  *testfake.FakeDocker
	state   *testfake.FakeState
	manager *Manager
}

func newTestSetup(t *testing.T) *testSetup {
	t.Helper()
	fd := new(testfake.FakeDocker)
	fs := testfake.NewFakeState()
	manager, err := newManager(t.TempDir(), fd, fs)
	assert.NotErr(t, err)
	return &testSetup{
		docker:  fd,
		state:   fs,
		manager: manager,
	}
}

func (ts *testSetup) setWorkspace(t *testing.T, id, status string, opts ...wsOption) string {
	t.Helper()
	basePath := t.TempDir()
	cfg := &wsConfig{}
	ws := &state.Workspace{
		ID:       id,
		Name:     "test",
		Status:   status,
		BasePath: basePath,
	}
	for _, opt := range opts {
		opt(t, ws, cfg)
	}
	if cfg.yaml == "" {
		cfg.yaml = "name: " + ws.Name + "\nsource:\n  local_path: " + t.TempDir()
	}
	writeConfig(t, basePath, cfg.yaml)
	ts.state.Data[id] = ws
	return basePath
}

func writeConfig(t *testing.T, basePath, content string) {
	t.Helper()
	path := filepath.Join(basePath, "config.yaml")
	assert.NotErr(t, os.WriteFile(path, []byte(content), 0o644))
}

type wsOption func(t *testing.T, ws *state.Workspace, cfg *wsConfig)

type wsConfig struct {
	yaml string
}

func withContainers(ids map[string]string) wsOption {
	return func(_ *testing.T, ws *state.Workspace, _ *wsConfig) {
		ws.ContainerIDs = ids
	}
}

func withAgent() wsOption {
	return withContainers(map[string]string{"agent": "ctr-1"})
}

func withAgentAndProxy() wsOption {
	return withContainers(map[string]string{"agent": "ctr-1", "proxy": "ctr-2"})
}

func withNetwork(id string) wsOption {
	return func(_ *testing.T, ws *state.Workspace, _ *wsConfig) {
		ws.NetworkID = id
	}
}

func withFullDocker() wsOption {
	return func(t *testing.T, ws *state.Workspace, cfg *wsConfig) {
		withAgentAndProxy()(t, ws, cfg)
		withNetwork("net-1")(t, ws, cfg)
	}
}

func withName(name string) wsOption {
	return func(_ *testing.T, ws *state.Workspace, _ *wsConfig) {
		ws.Name = name
	}
}

func withConfig(yaml string) wsOption {
	return func(_ *testing.T, _ *state.Workspace, cfg *wsConfig) {
		cfg.yaml = yaml
	}
}

func withLogDir() wsOption {
	return func(t *testing.T, ws *state.Workspace, _ *wsConfig) {
		testutil.MkdirAll(t, filepath.Join(ws.BasePath, "logs", "agent"))
	}
}
