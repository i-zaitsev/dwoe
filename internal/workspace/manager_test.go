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

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/docker"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	assert.NotNil(t, ts.manager)
}

func TestManager_Create(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)

	ws, err := ts.manager.Create(&config.Task{
		Name:   "test-task",
		Source: config.Source{LocalPath: t.TempDir()},
	})

	assert.NotErr(t, err)
	assert.NotZero(t, ws.ID)
	assert.Equal(t, ws.Name, "test-task")
	assert.Equal(t, ws.Status, "pending")
	assert.NotZero(t, ts.state.Data[ws.ID])
	for _, dir := range []string{"workspace", "logs/agent", "logs/proxy", ".docker", "config.yaml"} {
		assert.PathExists(t, filepath.Join(ws.BasePath, dir))
	}
}

func TestManager_Get(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "test-id", "testing", withConfig("description: test task"))

	t.Run("found_by_id", func(t *testing.T) {
		ws, err := ts.manager.Get("test-id")
		assert.NotErr(t, err)
		assert.Equal(t, ws.Status, "testing")
		assert.Equal(t, ws.Config.Description, "test task")
	})

	t.Run("not_found_by_id", func(t *testing.T) {
		_, err := ts.manager.Get("nonexistent")
		assert.Err(t, err)
	})
}

func TestManager_GetByName(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "some-id", "", withName("my-task"), withConfig("{}"))

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
	ts.setWorkspace(t, "test-1", "", withConfig("description: test-1"))
	ts.setWorkspace(t, "test-2", "", withConfig("description: test-2"))

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
	ts.setWorkspace(t, "ws-1", "pending")

	err := ts.manager.Start(context.Background(), "ws-1")

	assert.NotErr(t, err)
	assert.NoDiff(t, []string{
		"CreateNetwork",
		"CreateContainer", "StartContainer", // proxy
		"CreateContainer", "StartContainer", // agent
	}, ts.docker.Calls)
	assert.Equal(t, ts.state.Data["ws-1"].Status, "running")
	assert.HasKey(t, ts.state.Data["ws-1"].ContainerIDs, "proxy")
	assert.HasKey(t, ts.state.Data["ws-1"].ContainerIDs, "agent")
	assert.NotZero(t, ts.state.Data["ws-1"].NetworkID)
}

func TestManager_Start_Templates(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := ts.setWorkspace(t, "ws-1", "pending")

	err := ts.manager.Start(context.Background(), "ws-1")

	assert.NotErr(t, err)
	for _, name := range []string{
		filepath.Join("proxy", "squid.conf"),
		filepath.Join("proxy", "allowlist.txt"),
		"settings.json",
		"CLAUDE.md",
	} {
		assert.PathExists(t, filepath.Join(basePath, name))
	}
}

func TestManager_Start_ContainerConfig(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", "pending",
		withName("my-task"),
		withConfig(strings.Join([]string{
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
		}, "\n")))

	err := ts.manager.Start(context.Background(), "ws-1")

	wantAction := "CreateContainer"
	assert.NotErr(t, err)
	if !slices.Contains(ts.docker.Calls, wantAction) {
		t.Fatalf("%s not called", wantAction)
	}
}

func TestManager_Start_WithProxy(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", "pending",
		withName("proxy-test"),
		withConfig(strings.Join([]string{
			"name: proxy-test",
			"source:",
			"  local_path: /tmp",
			"network:",
			"  proxy:",
			"    port: 3128",
			"    base_allowlist:",
			"      - example.com",
		}, "\n")))

	err := ts.manager.Start(context.Background(), "ws-1")

	assert.NotErr(t, err)
	assert.NoDiff(t, []string{
		"CreateNetwork",
		"CreateContainer", "StartContainer", // proxy
		"CreateContainer", "StartContainer", // agent
	}, ts.docker.Calls)
	assert.HasKey(t, ts.state.Data["ws-1"].ContainerIDs, "proxy")
	assert.HasKey(t, ts.state.Data["ws-1"].ContainerIDs, "agent")
}

func TestManager_Start_WithoutProxy(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", "pending",
		withConfig("name: test\nsource:\n  local_path: /tmp\nno_proxy: true"),
	)

	err := ts.manager.Start(context.Background(), "ws-1")

	assert.NotErr(t, err)
	assert.NoKey(t, ts.state.Data["ws-1"].ContainerIDs, "proxy")
}

func TestManager_Stop_WithProxy(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", "running",
		withContainers(map[string]string{
			"agent": "ctr-agent",
			"proxy": "ctr-proxy",
		}),
	)

	err := ts.manager.Stop(context.Background(), "ws-1", time.Minute)

	assert.NotErr(t, err)
	assert.NoDiff(t, []string{"StopContainer", "StopContainer"}, ts.docker.Calls) // stops both containers
}

func TestManager_Stop(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", "running", withAgent())

	err := ts.manager.Stop(context.Background(), "ws-1", time.Minute)

	assert.NotErr(t, err)
	assert.NoDiff(t, []string{"StopContainer"}, ts.docker.Calls)
	assert.Equal(t, ts.state.Data["ws-1"].Status, "stopped")
}

func TestManager_Logs(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", "running", withAgent())

	rc, err := ts.manager.Logs(context.Background(), "ws-1", true)

	assert.NotErr(t, err)
	assert.NotNil(t, rc)
	_ = rc.Close()
	assert.NoDiff(t, []string{"ContainerLogs"}, ts.docker.Calls)
}

func TestManager_Wait(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", "running", withAgent())

	_, err := ts.manager.Wait(context.Background(), "ws-1")
	assert.NotErr(t, err)

	assert.NoDiff(t, []string{"WaitContainer"}, ts.docker.Calls)
	assert.Equal(t, ts.state.Data["ws-1"].Status, "completed")
	assert.NotNil(t, ts.state.Data["ws-1"].FinishedAt)
}

func TestManager_Sync_ContainerExit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		inspectFn  func(context.Context, string) (docker.ContainerInfo, error)
		wantStatus string
	}{
		{
			name: "exited_success",
			inspectFn: func(context.Context, string) (docker.ContainerInfo, error) {
				return docker.ContainerInfo{Status: "exited", ExitCode: 0}, nil
			},
			wantStatus: "completed",
		},
		{
			name: "exited_failure",
			inspectFn: func(context.Context, string) (docker.ContainerInfo, error) {
				return docker.ContainerInfo{Status: "exited", ExitCode: 1}, nil
			},
			wantStatus: "failed",
		},
		{
			name: "container_gone",
			inspectFn: func(context.Context, string) (docker.ContainerInfo, error) {
				return docker.ContainerInfo{}, errors.New("container not found")
			},
			wantStatus: "failed",
		},
	}
	wantCalls := []string{"InspectContainer", "ContainerLogs", "RemoveContainer", "RemoveContainer", "RemoveNetwork"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := newTestSetup(t)
			ts.setWorkspace(t, "ws-1", "running", withFullDocker())
			ts.docker.InspectContainerFn = tt.inspectFn

			assert.NotErr(t, ts.manager.Sync(context.Background(), "ws-1"))

			ws := ts.state.Data["ws-1"]
			assert.Equal(t, ws.Status, tt.wantStatus)
			assert.NotNil(t, ws.FinishedAt)
			assert.Nil(t, ws.ContainerIDs)
			assert.Zero(t, ws.NetworkID)
			assert.NoDiff(t, wantCalls, ts.docker.Calls)
		})
	}
}

func TestManager_Sync_StillRunning(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", "running", withAgent())

	err := ts.manager.Sync(context.Background(), "ws-1")

	assert.NotErr(t, err)
	ws := ts.state.Data["ws-1"]
	assert.Equal(t, ws.Status, "running")
	assert.Nil(t, ws.FinishedAt)
}

func TestManager_Sync_NotRunning(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", "completed")

	err := ts.manager.Sync(context.Background(), "ws-1")

	assert.NotErr(t, err)
	assert.Zero(t, len(ts.docker.Calls))
}

func TestManager_Destroy(t *testing.T) {
	t.Parallel()

	wantCalls := []string{"RemoveContainer", "RemoveContainer", "RemoveNetwork"}

	for _, keepDir := range []bool{true, false} {
		t.Run(fmt.Sprintf("keep-dir=%v", keepDir), func(t *testing.T) {
			t.Parallel()
			ts := newTestSetup(t)
			basePath := ts.setWorkspace(t, "ws-1", "stopped", withFullDocker())

			err := ts.manager.Destroy(
				context.Background(),
				"ws-1",
				DestroyOpts{KeepDir: keepDir},
			)

			assert.NotErr(t, err)
			assert.NoDiff(t, wantCalls, ts.docker.Calls)
			assert.Nil(t, ts.state.Data["ws-1"])
			if !keepDir {
				assert.NoPathExists(t, basePath)
			}
		})
	}
}

func TestManager_Cleanup(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := ts.setWorkspace(t, "ws-1", "completed", withFullDocker(), withLogDir())
	wantCalls := []string{"ContainerLogs", "RemoveContainer", "RemoveContainer", "RemoveNetwork"}

	err := ts.manager.Cleanup(context.Background(), "ws-1")

	assert.NotErr(t, err)
	assert.NoDiff(t, wantCalls, ts.docker.Calls)
	assert.NotNil(t, ts.state.Data["ws-1"])
	assert.Nil(t, ts.state.Data["ws-1"].ContainerIDs)
	assert.Zero(t, ts.state.Data["ws-1"].NetworkID)
	assert.PathExists(t, basePath)
}

func TestManager_Cleanup_SavesLogs(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := ts.setWorkspace(t, "ws-1", "completed", withAgent(), withLogDir())
	wantLogs := "line 1\nline 2\n"
	ts.docker.ContainerLogsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(wantLogs)), nil
	}

	err := ts.manager.Cleanup(context.Background(), "ws-1")

	assert.NotErr(t, err)
	data, err := os.ReadFile(filepath.Join(basePath, "logs", "agent", savedLogFile))
	assert.NotErr(t, err)
	assert.Equal(t, string(data), wantLogs)
}

func TestManager_Logs_FallbackToFile(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := ts.setWorkspace(t, "ws-1", "completed", withLogDir())
	testutil.WriteFile(t, filepath.Join(basePath, "logs", "agent", savedLogFile), "saved output\n")

	rc, err := ts.manager.Logs(context.Background(), "ws-1", false)

	assert.NotErr(t, err)
	data, err := io.ReadAll(rc)
	assert.NotErr(t, err)
	_ = rc.Close()
	assert.Equal(t, string(data), "saved output\n")
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
	assert.Equal(t, ws2.Name, "dup-test-1")
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
			assert.NotErr(t, ts.manager.UpdateStatus(ws.ID, tt.status))
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
			assert.Equal(t, got.ID, ws.ID)
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
	ts.setWorkspace(t, "abc-111-full-id", "running", withName("ws-one"))

	ws, err := ts.manager.ResolvePrefix("abc-111")

	assert.NotErr(t, err)
	assert.Equal(t, ws.ID, "abc-111-full-id")
}

func TestManager_ResolvePrefix_Ambiguous(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	for _, id := range []string{"abc-111", "abc-222"} {
		ts.setWorkspace(t, id, "running", withName(id))
	}

	_, err := ts.manager.ResolvePrefix("abc")

	assert.ErrAs[*state.AmbiguousMatchError](t, err)
}

func TestManager_ResolvePrefix_NotFound(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)

	_, err := ts.manager.ResolvePrefix("zzz")

	assert.ErrAs[*state.NotFoundError](t, err)
}

func TestManager_Resolve_PrefixFallback(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "abcdef-1234-5678", "completed", withName("my-ws"))

	ws, err := ts.manager.Resolve("abcdef")

	assert.NotErr(t, err)
	assert.Equal(t, ws.ID, "abcdef-1234-5678")
}

func TestManager_Diff(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	basePath := ts.setWorkspace(t, "ws-1", "completed")

	wsDir := filepath.Join(basePath, "workspace")
	testutil.MkdirAll(t, wsDir)
	mustGit(t, "init", wsDir, "--initial-branch", "main")
	mustGit(t, "-C", wsDir, "config", "user.name", "Agent")
	mustGit(t, "-C", wsDir, "config", "user.email", "agent@test.dev")
	testutil.WriteFile(t, filepath.Join(wsDir, "base.txt"), "base")
	mustGit(t, "-C", wsDir, "add", ".")
	mustGit(t, "-C", wsDir, "commit", "-m", "initial")
	addCommit(t, wsDir, "feature.go", "package main", "add feature")

	info, err := ts.manager.Diff("ws-1")
	assert.NotErr(t, err)
	assert.Equal(t, len(info.Commits), 1)
	assert.Equal(t, info.Commits[0].Message, "add feature")
	assert.Contains(t, info.Stat, "feature.go")
	assert.Contains(t, info.Diff, "package main")
}

func TestManager_Sync_OrphanedRunning(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", StatusRunning)
	ts.state.Data["ws-1"].ContainerIDs = nil // orphaned state: running w/o container

	err := ts.manager.Sync(context.Background(), "ws-1")

	assert.NotErr(t, err)
	assert.Equal(t, ts.state.Data["ws-1"].Status, StatusFailed)
	assert.NotNil(t, ts.state.Data["ws-1"].FinishedAt)
}

func TestManager_Stop_MissingContainers(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "ws-1", StatusRunning)
	ts.state.Data["ws-1"].ContainerIDs = nil // orphaned state: running w/o container

	err := ts.manager.Stop(context.Background(), "ws-1", time.Minute)

	assert.NotErr(t, err)
	assert.Equal(t, ts.state.Data["ws-1"].Status, StatusStopped)
}

func TestManager_FindOrCreate_CreatesNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		policy config.ContinuePolicy
	}{
		{"default", config.ContinuePolicyDefault},
		{"restart", config.ContinuePolicyRestart},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := newTestSetup(t)

			ws, err := ts.manager.FindOrCreate(context.Background(), &config.Task{
				Name:           "new-task",
				ContinuePolicy: tt.policy,
				Source:         config.Source{LocalPath: t.TempDir()},
			})

			assert.NotErr(t, err)
			assert.Equal(t, ws.Status, StatusPending)
		})
	}
}

func TestManager_FindOrCreate_RestartWithExisting(t *testing.T) {
	t.Parallel()
	ts := newTestSetup(t)
	ts.setWorkspace(t, "old-ws", StatusCompleted, withName("my-task"))

	ws, err := ts.manager.FindOrCreate(context.Background(), &config.Task{
		Name:           "my-task",
		ContinuePolicy: config.ContinuePolicyRestart,
		Source:         config.Source{LocalPath: t.TempDir()},
	})

	assert.NotErr(t, err)
	assert.NotEqual(t, ws.ID, "old-ws")
}

func TestManager_FindOrCreate_Resume(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		status     string
		wantStatus string
	}{
		{"completed", StatusCompleted, StatusStopped},
		{"failed", StatusFailed, StatusStopped},
		{"stopped", StatusStopped, StatusStopped},
		{"running", StatusRunning, StatusStopped},
		{"pending", StatusPending, StatusPending},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := newTestSetup(t)
			ts.setWorkspace(t, "ws-1", tt.status, withName("my-task"))

			ws, err := ts.manager.FindOrCreate(context.Background(), &config.Task{
				Name:           "my-task",
				ContinuePolicy: config.ContinuePolicyResume,
				Source:         config.Source{LocalPath: t.TempDir()},
			})

			assert.NotErr(t, err)
			assert.Equal(t, ws.ID, "ws-1")
			assert.Equal(t, ws.Status, tt.wantStatus)
		})
	}
}

func TestManager_FindOrCreate_ResumeErrors(t *testing.T) {
	t.Parallel()

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		ts := newTestSetup(t)

		_, err := ts.manager.FindOrCreate(context.Background(), &config.Task{
			Name:           "nonexistent",
			ContinuePolicy: config.ContinuePolicyResume,
			Source:         config.Source{LocalPath: t.TempDir()},
		})

		assert.ErrAs[*state.NotFoundError](t, err)
	})

	t.Run("no_name", func(t *testing.T) {
		t.Parallel()
		ts := newTestSetup(t)

		_, err := ts.manager.FindOrCreate(context.Background(), &config.Task{
			ContinuePolicy: config.ContinuePolicyResume,
			Source:         config.Source{LocalPath: t.TempDir()},
		})

		assert.ErrIs(t, err, errContinueRequiresName)
	})
}
