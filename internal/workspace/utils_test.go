// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package workspace

import (
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

type wsOption func(t *testing.T, ws *state.Workspace)

func (ts *testSetup) setWorkspace(t *testing.T, id, status string, opts ...wsOption) string {
	t.Helper()
	dir := t.TempDir()
	ws := testfake.CreateWorkspace(t, dir, id, "test", status)
	for _, opt := range opts {
		opt(t, ws)
	}
	ts.state.Data[id] = ws
	return ws.BasePath
}

func writeConfig(t *testing.T, basePath, content string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(basePath, "config.yaml"), content)
}

func withContainers(ids map[string]string) wsOption {
	return func(_ *testing.T, ws *state.Workspace) {
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
	return func(_ *testing.T, ws *state.Workspace) {
		ws.NetworkID = id
	}
}

func withFullDocker() wsOption {
	return func(t *testing.T, ws *state.Workspace) {
		withAgentAndProxy()(t, ws)
		withNetwork("net-1")(t, ws)
	}
}

func withName(name string) wsOption {
	return func(_ *testing.T, ws *state.Workspace) {
		ws.Name = name
	}
}

func withConfig(yaml string) wsOption {
	return func(t *testing.T, ws *state.Workspace) {
		writeConfig(t, ws.BasePath, yaml)
	}
}

func withLogDir() wsOption {
	return func(t *testing.T, ws *state.Workspace) {
		testutil.MkdirAll(t, filepath.Join(ws.BasePath, "logs", "agent"))
	}
}
