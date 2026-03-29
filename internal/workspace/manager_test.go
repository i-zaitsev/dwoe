// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/docker"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/testfake"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	if ts.manager == nil {
		t.Fatal("expected manager, got nil")
	}
}

func TestManager_Create(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)

	ws, err := ts.manager.Create(&config.Task{
		Name:   "test-task",
		Source: config.Source{LocalPath: t.TempDir()},
	})
	assert.NotErr(t, err)

	if ws.ID == "" {
		t.Error("workspace ID should not be empty")
	}
	assert.Equal(t, ws.Name, "test-task")
	assert.Equal(t, ws.Status, "pending")

	dirs := []string{"workspace", "logs/agent", "logs/proxy", ".docker"}
	for _, dir := range dirs {
		path := filepath.Join(ws.BasePath, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("workspace directory %q should exist", path)
		}
	}

	path := filepath.Join(ws.BasePath, "config.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("workspace config file %q should exist", path)
	}

	if ts.state.Data[ws.ID] == nil {
		t.Error("state not saved")
	}
}

func TestManager_Get(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "description: test task")
	ts.state.Data["test-id"] = &state.Workspace{
		ID:       "test-id",
		BasePath: basePath,
		Status:   "testing",
	}

	ws, err := ts.manager.Get("test-id")

	t.Run("found_by_id", func(t *testing.T) {
		assert.NotErr(t, err)
		assert.Equal(t, ws.Status, "testing")
		assert.Equal(t, ws.Config.Description, "test task")
	})

	t.Run("not_found_by_id", func(t *testing.T) {
		_, err = ts.manager.Get("nonexistent")
		assert.Err(t, err)
	})
}

func TestManager_GetByName(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "{}")
	ts.state.Data["some-id"] = &state.Workspace{
		ID:       "some-id",
		Name:     "my-task",
		BasePath: basePath,
	}

	t.Run("found_by_name", func(t *testing.T) {
		ws, err := ts.manager.GetByName("my-task")
		assert.NotErr(t, err)
		assert.Equal(t, ws.ID, "some-id")
	})

	t.Run("not_found_by_name", func(t *testing.T) {
		_, err := ts.manager.GetByName("nonexistent")
		assert.Err(t, err)
	})
}

func TestManager_List(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	for _, name := range []string{"test-1", "test-2"} {
		basePath := t.TempDir()
		writeConfig(t, basePath, "description: "+name)
		ts.state.Data[name] = &state.Workspace{
			ID:       name,
			BasePath: basePath,
		}
	}

	ws, err := ts.manager.List()

	assert.NotErr(t, err)
	assert.Equal(t, len(ws), 2)
	ids := map[string]bool{}
	for _, w := range ws {
		ids[w.ID] = true
	}
	if !(ids["test-1"] && ids["test-2"]) {
		t.Errorf("expected workspaces test-1 and test-2, got %v", ids)
	}
}

func TestManager_Start(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:       "ws-1",
		Name:     "test",
		Status:   "pending",
		BasePath: basePath,
	}

	err := ts.manager.Start(context.Background(), "ws-1")

	assert.NotErr(t, err)
	want := []string{
		"CreateNetwork",
		"CreateContainer", "StartContainer",
		"CreateContainer", "StartContainer",
	}
	got := ts.docker.Calls
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
	assert.Equal(t, ts.state.Data["ws-1"].Status, "running")
	if _, ok := ts.state.Data["ws-1"].ContainerIDs["proxy"]; !ok {
		t.Error("proxy container ID should be set")
	}
	if _, ok := ts.state.Data["ws-1"].ContainerIDs["agent"]; !ok {
		t.Error("agent container ID should be set")
	}
	if ts.state.Data["ws-1"].NetworkID == "" {
		t.Error("NetworkID should be set")
	}
}

func TestManager_Start_Templates(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:       "ws-1",
		Name:     "test",
		Status:   "pending",
		BasePath: basePath,
	}

	err := ts.manager.Start(context.Background(), "ws-1")

	assert.NotErr(t, err)
	for _, name := range []string{
		filepath.Join("proxy", "squid.conf"),
		filepath.Join("proxy", "allowlist.txt"),
		"settings.json",
		"CLAUDE.md",
	} {
		path := filepath.Join(basePath, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected template %s to exist: %v", name, err)
		}
	}
}

func TestManager_Start_ContainerConfig(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	cfgYAML := strings.Join([]string{
		"name: my-task",
		"source:",
		"  local_path: /tmp",
		"agent:",
		"  image: custom:v1",
		"  model: best-model",
		"  max_turns: 50",
		"resources:",
		"  cpu: '2'",
		"  memory: 4G",
	}, "\n")
	writeConfig(t, basePath, cfgYAML)
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:       "ws-1",
		Name:     "my-task",
		Status:   "pending",
		BasePath: basePath,
	}

	err := ts.manager.Start(context.Background(), "ws-1")

	assert.NotErr(t, err)

	if !slices.Contains(ts.docker.Calls, "CreateContainer") {
		t.Fatal("CreateContainer not called")
	}
}

func TestManager_Start_WithProxy(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	cfgYAML := strings.Join([]string{
		"name: proxy-test",
		"source:",
		"  local_path: /tmp",
		"network:",
		"  proxy:",
		"    port: 3128",
		"    base_allowlist:",
		"      - example.com",
	}, "\n")
	writeConfig(t, basePath, cfgYAML)
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:       "ws-1",
		Name:     "proxy-test",
		Status:   "pending",
		BasePath: basePath,
	}

	err := ts.manager.Start(context.Background(), "ws-1")

	assert.NotErr(t, err)
	want := []string{
		"CreateNetwork",
		"CreateContainer", "StartContainer",
		"CreateContainer", "StartContainer",
	}
	got := ts.docker.Calls
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
	if _, ok := ts.state.Data["ws-1"].ContainerIDs["proxy"]; !ok {
		t.Error("proxy container ID should be set")
	}
	if _, ok := ts.state.Data["ws-1"].ContainerIDs["agent"]; !ok {
		t.Error("agent container ID should be set")
	}
}

func TestManager_Start_WithoutProxy(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp\nno_proxy: true")
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:       "ws-1",
		Name:     "test",
		Status:   "pending",
		BasePath: basePath,
	}

	err := ts.manager.Start(context.Background(), "ws-1")

	assert.NotErr(t, err)
	if _, ok := ts.state.Data["ws-1"].ContainerIDs["proxy"]; ok {
		t.Error("proxy container ID should not be set when proxy is disabled")
	}
}

func TestManager_Stop_WithProxy(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "running",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-agent", "proxy": "ctr-proxy"},
	}

	err := ts.manager.Stop(context.Background(), "ws-1", time.Minute)

	assert.NotErr(t, err)
	want := []string{"StopContainer", "StopContainer"}
	got := ts.docker.Calls
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
}

func TestManager_Stop(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "running",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-1"},
	}

	err := ts.manager.Stop(context.Background(), "ws-1", time.Minute)

	assert.NotErr(t, err)
	want := []string{"StopContainer"}
	got := ts.docker.Calls
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
	assert.Equal(t, ts.state.Data["ws-1"].Status, "stopped")
}

func TestManager_Logs(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "running",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-1"},
	}

	rc, err := ts.manager.Logs(context.Background(), "ws-1", true)

	assert.NotErr(t, err)
	if rc == nil {
		t.Fatal("expected logs reader, got nil")
	}
	_ = rc.Close()
	want := []string{"ContainerLogs"}
	got := ts.docker.Calls
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
}

func TestManager_Wait(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: "+t.TempDir())
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "running",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-1"},
	}

	if _, err := ts.manager.Wait(context.Background(), "ws-1"); err != nil {
		t.Fatal(err)
	}

	want := []string{"WaitContainer"}
	got := ts.docker.Calls
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
	assert.Equal(t, ts.state.Data["ws-1"].Status, "completed")
	if ts.state.Data["ws-1"].FinishedAt == nil {
		t.Error("FinishedAt should be set after Wait")
	}
}

func TestManager_Sync_ContainerExited(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: "+t.TempDir())
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "running",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-1", "proxy": "ctr-2"},
		NetworkID:    "net-1",
	}
	ts.docker.InspectContainerFn = func(context.Context, string) (docker.ContainerInfo, error) {
		return docker.ContainerInfo{Status: "exited", ExitCode: 0}, nil
	}

	if err := ts.manager.Sync(context.Background(), "ws-1"); err != nil {
		t.Fatal(err)
	}

	ws := ts.state.Data["ws-1"]
	assert.Equal(t, ws.Status, "completed")
	if ws.FinishedAt == nil {
		t.Error("FinishedAt should be set")
	}
	if ws.ContainerIDs != nil {
		t.Error("ContainerIDs should be cleared")
	}
	assert.Zero(t, ws.NetworkID)
	wantCalls := []string{"InspectContainer", "ContainerLogs", "RemoveContainer", "RemoveContainer", "RemoveNetwork"}
	if diff := cmp.Diff(wantCalls, ts.docker.Calls); diff != "" {
		t.Errorf("docker calls (-want, +got):\n%s", diff)
	}
}

func TestManager_Sync_ContainerFailed(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: "+t.TempDir())
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "running",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-1", "proxy": "ctr-2"},
		NetworkID:    "net-1",
	}
	ts.docker.InspectContainerFn = func(context.Context, string) (docker.ContainerInfo, error) {
		return docker.ContainerInfo{Status: "exited", ExitCode: 1}, nil
	}

	if err := ts.manager.Sync(context.Background(), "ws-1"); err != nil {
		t.Fatal(err)
	}

	ws := ts.state.Data["ws-1"]
	assert.Equal(t, ws.Status, "failed")
	if ws.FinishedAt == nil {
		t.Error("FinishedAt should be set")
	}
	if ws.ContainerIDs != nil {
		t.Error("ContainerIDs should be cleared")
	}
	assert.Zero(t, ws.NetworkID)
	wantCalls := []string{"InspectContainer", "ContainerLogs", "RemoveContainer", "RemoveContainer", "RemoveNetwork"}
	if diff := cmp.Diff(wantCalls, ts.docker.Calls); diff != "" {
		t.Errorf("docker calls (-want, +got):\n%s", diff)
	}
}

func TestManager_Sync_ContainerGone(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: "+t.TempDir())
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "running",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-1", "proxy": "ctr-2"},
		NetworkID:    "net-1",
	}
	ts.docker.InspectContainerFn = func(context.Context, string) (docker.ContainerInfo, error) {
		return docker.ContainerInfo{}, errors.New("container not found")
	}

	if err := ts.manager.Sync(context.Background(), "ws-1"); err != nil {
		t.Fatal(err)
	}

	ws := ts.state.Data["ws-1"]
	assert.Equal(t, ws.Status, "failed")
	if ws.FinishedAt == nil {
		t.Error("FinishedAt should be set")
	}
	if ws.ContainerIDs != nil {
		t.Error("ContainerIDs should be cleared")
	}
	assert.Zero(t, ws.NetworkID)
	wantCalls := []string{"InspectContainer", "ContainerLogs", "RemoveContainer", "RemoveContainer", "RemoveNetwork"}
	if diff := cmp.Diff(wantCalls, ts.docker.Calls); diff != "" {
		t.Errorf("docker calls (-want, +got):\n%s", diff)
	}
}

func TestManager_Sync_StillRunning(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: "+t.TempDir())
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "running",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-1"},
	}

	if err := ts.manager.Sync(context.Background(), "ws-1"); err != nil {
		t.Fatal(err)
	}

	ws := ts.state.Data["ws-1"]
	assert.Equal(t, ws.Status, "running")
	if ws.FinishedAt != nil {
		t.Error("FinishedAt should not be set")
	}
}

func TestManager_Sync_NotRunning(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: "+t.TempDir())
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:       "ws-1",
		Name:     "test",
		Status:   "completed",
		BasePath: basePath,
	}

	if err := ts.manager.Sync(context.Background(), "ws-1"); err != nil {
		t.Fatal(err)
	}

	assert.Zero(t, len(ts.docker.Calls))
}

func TestManager_Destroy(t *testing.T) {
	t.Parallel()
	for _, keepDir := range []bool{true, false} {

		t.Run(fmt.Sprintf("keep-dir=%v", keepDir), func(t *testing.T) {
			t.Parallel()

			ts := newTestSetup(t)
			basePath := t.TempDir()
			writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")

			ts.state.Data["ws-1"] = &state.Workspace{
				ID:           "ws-1",
				Name:         "test",
				Status:       "stopped",
				BasePath:     basePath,
				ContainerIDs: map[string]string{"agent": "ctr-1", "proxy": "ctr-2"},
				NetworkID:    "net-1",
			}

			err := ts.manager.Destroy(
				context.Background(),
				"ws-1",
				DestroyOpts{KeepDir: keepDir},
			)

			if err != nil {
				t.Fatal(err)
			}

			want := []string{"RemoveContainer", "RemoveContainer", "RemoveNetwork"}
			got := ts.docker.Calls
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
			if ts.state.Data["ws-1"] != nil {
				t.Error("workspace should be deleted from state")
			}

			if !keepDir {
				if _, errDel := os.Stat(basePath); !os.IsNotExist(errDel) {
					t.Errorf("keepDir=false, workspace directory should be deleted")
				}
			}
		})
	}
}

func TestManager_Cleanup(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")
	testutil.MkdirAll(t, filepath.Join(basePath, "logs", "agent"))
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "completed",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-1", "proxy": "ctr-2"},
		NetworkID:    "net-1",
	}

	err := ts.manager.Cleanup(context.Background(), "ws-1")

	assert.NotErr(t, err)
	want := []string{"ContainerLogs", "RemoveContainer", "RemoveContainer", "RemoveNetwork"}
	got := ts.docker.Calls
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
	if ts.state.Data["ws-1"] == nil {
		t.Fatal("state entry should be preserved")
	}
	if ts.state.Data["ws-1"].ContainerIDs != nil {
		t.Error("ContainerIDs should be cleared")
	}
	assert.Zero(t, ts.state.Data["ws-1"].NetworkID)
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		t.Error("workspace directory should be preserved")
	}
}

func TestManager_Cleanup_SavesLogs(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")
	testutil.MkdirAll(t, filepath.Join(basePath, "logs", "agent"))
	ts.docker.ContainerLogsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("line 1\nline 2\n")), nil
	}
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "completed",
		BasePath:     basePath,
		ContainerIDs: map[string]string{"agent": "ctr-1"},
	}

	err := ts.manager.Cleanup(context.Background(), "ws-1")
	assert.NotErr(t, err)

	logPath := filepath.Join(basePath, "logs", "agent", savedLogFile)
	data, err := os.ReadFile(logPath)
	assert.NotErr(t, err)
	assert.Equal(t, string(data), "line 1\nline 2\n")
}

func TestManager_Logs_FallbackToFile(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")
	logDir := filepath.Join(basePath, "logs", "agent")
	testutil.MkdirAll(t, logDir)
	testutil.WriteFile(t, filepath.Join(logDir, savedLogFile), "saved output\n")
	ts.state.Data["ws-1"] = &state.Workspace{
		ID:           "ws-1",
		Name:         "test",
		Status:       "completed",
		BasePath:     basePath,
		ContainerIDs: nil,
	}

	rc, err := ts.manager.Logs(context.Background(), "ws-1", false)
	assert.NotErr(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	assert.NotErr(t, err)
	assert.Equal(t, string(data), "saved output\n")
}

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

func writeConfig(t *testing.T, basePath, content string) {
	t.Helper()
	path := filepath.Join(basePath, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestManager_CreateDuplicateName(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	src := t.TempDir()
	ws1, err := ts.manager.Create(&config.Task{
		Name:   "dup-test",
		Source: config.Source{LocalPath: src},
	})
	assert.NotErr(t, err)
	ws2, err := ts.manager.Create(&config.Task{
		Name:   "dup-test",
		Source: config.Source{LocalPath: src},
	})
	assert.NotErr(t, err)
	assert.Equal(t, ws1.Name, "dup-test")
	if ws2.Name == "dup-test" {
		t.Error("second workspace should have a different name")
	}
}

func TestManager_CreateCopiesPromptFile(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	srcDir := t.TempDir()
	promptFile := filepath.Join(t.TempDir(), "task-prompt.md")
	testutil.WriteFile(t, promptFile, "Do the thing")

	ws, err := ts.manager.Create(&config.Task{
		Name: "prompt-test",
		Source: config.Source{
			LocalPath:  srcDir,
			PromptFile: promptFile,
		},
	})
	assert.NotErr(t, err)

	data, err := os.ReadFile(filepath.Join(ws.BasePath, "workspace", "task-prompt.md"))
	assert.NotErr(t, err)
	assert.Equal(t, string(data), "Do the thing")
}

func TestManager_ResolveCompleted(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)

	ws, err := ts.manager.Create(&config.Task{
		Name:   "resolve-test",
		Source: config.Source{LocalPath: t.TempDir()},
	})
	assert.NotErr(t, err)

	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{"pending", "pending", true},
		{"running", "running", true},
		{"completed", "completed", false},
		{"failed", "failed", false},
		{"stopped", "stopped", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ts.manager.UpdateStatus(ws.ID, tt.status); err != nil {
				t.Fatal(err)
			}
			got, err := ts.manager.ResolveCompleted(ws.ID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for status %q", tt.status)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != ws.ID {
				t.Errorf("ID = %q, want %q", got.ID, ws.ID)
			}
		})
	}
}

func TestManager_ResolveCompleted_NotFound(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	_, err := ts.manager.ResolveCompleted("no-such-id")
	assert.Err(t, err)
}

func TestManager_ResolvePrefix_UniqueMatch(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")
	ts.state.Data["abc-111-full-id"] = &state.Workspace{
		ID:       "abc-111-full-id",
		Name:     "ws-one",
		BasePath: basePath,
		Status:   "running",
	}

	ws, err := ts.manager.ResolvePrefix("abc-111")
	assert.NotErr(t, err)
	assert.Equal(t, ws.ID, "abc-111-full-id")
}

func TestManager_ResolvePrefix_Ambiguous(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	for _, id := range []string{"abc-111", "abc-222"} {
		bp := t.TempDir()
		writeConfig(t, bp, "name: "+id+"\nsource:\n  local_path: /tmp")
		ts.state.Data[id] = &state.Workspace{
			ID:       id,
			Name:     id,
			BasePath: bp,
			Status:   "running",
		}
	}

	_, err := ts.manager.ResolvePrefix("abc")
	testutil.WantErrAs[*state.AmbiguousMatchError](t, err)
}

func TestManager_ResolvePrefix_NotFound(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)

	_, err := ts.manager.ResolvePrefix("zzz")
	testutil.WantErrAs[*state.NotFoundError](t, err)
}

func TestManager_Resolve_PrefixFallback(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: my-ws\nsource:\n  local_path: /tmp")
	ts.state.Data["abcdef-1234-5678"] = &state.Workspace{
		ID:       "abcdef-1234-5678",
		Name:     "my-ws",
		BasePath: basePath,
		Status:   "completed",
	}

	ws, err := ts.manager.Resolve("abcdef")
	assert.NotErr(t, err)
	assert.Equal(t, ws.ID, "abcdef-1234-5678")
}

func TestStatusConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"pending", StatusPending, "pending"},
		{"running", StatusRunning, "running"},
		{"stopped", StatusStopped, "stopped"},
		{"completed", StatusCompleted, "completed"},
		{"failed", StatusFailed, "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.got, tt.want)
		})
	}
}

func TestManager_Diff(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := t.TempDir()
	writeConfig(t, basePath, "name: test\nsource:\n  local_path: /tmp")

	wsDir := filepath.Join(basePath, "workspace")
	testutil.MkdirAll(t, wsDir)
	mustGit(t, "init", wsDir, "--initial-branch", "main")
	mustGit(t, "-C", wsDir, "config", "user.name", "Agent")
	mustGit(t, "-C", wsDir, "config", "user.email", "agent@test.dev")
	testutil.WriteFile(t, filepath.Join(wsDir, "base.txt"), "base")
	mustGit(t, "-C", wsDir, "add", ".")
	mustGit(t, "-C", wsDir, "commit", "-m", "initial")
	testutil.WriteFile(t, filepath.Join(wsDir, "feature.go"), "package main")
	mustGit(t, "-C", wsDir, "add", ".")
	mustGit(t, "-C", wsDir, "commit", "-m", "add feature")

	ts.state.Data["ws-1"] = &state.Workspace{
		ID:       "ws-1",
		Name:     "test",
		Status:   "completed",
		BasePath: basePath,
	}

	info, err := ts.manager.Diff("ws-1")
	assert.NotErr(t, err)
	assert.Equal(t, len(info.Commits), 1)
	assert.Equal(t, info.Commits[0].Message, "add feature")
	assert.Contains(t, info.Stat, "feature.go")
	assert.Contains(t, info.Diff, "package main")
}
