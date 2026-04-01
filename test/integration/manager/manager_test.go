// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build integration

package manager_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/docker"
	"github.com/i-zaitsev/dwoe/internal/log"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/testutil"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

const basicImage = "dwoe-integration-test:latest"

var testLogs bytes.Buffer
var testClient *docker.Client

func fatal(err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func TestMain(m *testing.M) {
	var code int
	func() {
		opts := log.DefaultOpts()
		opts.Writer = &testLogs
		log.Setup(opts)

		client, err := docker.NewClient()
		fatal(err)

		testClient = client
		defer testClient.Close()

		ctx := context.Background()
		fatal(testClient.Ping(ctx))
		fatal(testClient.BuildImage(ctx, "../testdata/Dockerfile.basic", basicImage, os.Stderr))

		code = m.Run()
	}()
	os.Exit(code)
}

func TestManager_Lifecycle(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	store := state.NewStore(dataDir)

	manager, err := workspace.NewManagerWith(dataDir, testClient, store)
	assert.NotErr(t, err)

	cfg := testConfig(t)
	testutil.WriteFile(t, filepath.Join(cfg.Source.LocalPath, "hello.txt"), "hi")

	ws, err := manager.Create(cfg)
	assert.NotErr(t, err)
	if ws.Status != "pending" {
		t.Fatalf("workspace status is not pending, got: %s", ws.Status)
	}
	if !testutil.FileExists(ws.BasePath) {
		t.Fatalf("workspace base path does not exist: %s", ws.BasePath)
	}
	if !testutil.FileExists(filepath.Join(ws.BasePath, "workspace", "hello.txt")) {
		t.Fatalf("source file not copied into workspace")
	}

	assert.NotErr(t, manager.Start(ctx, ws.ID))

	ws2, err := manager.Get(ws.ID)
	assert.NotErr(t, err)
	if ws2.Status != "running" {
		t.Fatalf("workspace status is not running, got: %s", ws2.Status)
	}
	if !testutil.FileExists(filepath.Join(ws.BasePath, "CLAUDE.md")) {
		t.Fatalf("CLAUDE.md file does not exist in workspace base path: %s", ws.BasePath)
	}

	code, err := manager.Wait(ctx, ws.ID)
	assert.NotErr(t, err)
	if code != 0 {
		t.Fatalf("workspace exited with non-zero code: %d", code)
	}

	assert.NotErr(t, manager.Destroy(ctx, ws.ID, workspace.DestroyOpts{KeepDir: false}))
	_, err = manager.Get(ws.ID)
	assert.Err(t, err)
}

func TestManager_InvalidConfig(t *testing.T) {
	dataDir := t.TempDir()
	store := state.NewStore(dataDir)
	manager, err := workspace.NewManagerWith(dataDir, testClient, store)
	assert.NotErr(t, err)
	_, err = manager.Create(&config.Task{})
	assert.Err(t, err)
}

func TestManager_DestroyCleanup(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	store := state.NewStore(dataDir)

	manager, err := workspace.NewManagerWith(dataDir, testClient, store)
	assert.NotErr(t, err)

	ws, err := manager.Create(testConfig(t))
	assert.NotErr(t, err)

	assert.NotErr(t, manager.Start(ctx, ws.ID))
	assert.NotErr(t, manager.Destroy(ctx, ws.ID, workspace.DestroyOpts{}))

	if testutil.FileExists(ws.BasePath) {
		t.Fatalf("workspace base path still exists: %s", ws.BasePath)
	}
	_, err = manager.Get(ws.ID)
	assert.Err(t, err)
}

func testConfig(t *testing.T) *config.Task {
	t.Helper()
	return &config.Task{
		Name:        "integration-test-task",
		Description: "Integration test task",
		Source: config.Source{
			LocalPath: t.TempDir(),
		},
		Agent: config.Agent{
			Image:    basicImage,
			Model:    "claude-sonnet-4-6",
			MaxTurns: 999,
			EnvVars:  nil,
			Permissions: []string{
				"Bash(ls)",
				"Read(SPEC.md)",
			},
		},
		Git: config.GitUser{
			Name:  "Test User",
			Email: "test@test.com",
		},
		Network: config.Network{
			Proxy: config.Proxy{
				Port: 8080,
				AllowList: []string{
					"example.com",
					"*.go.dev",
				},
			},
			Name: fmt.Sprintf("inttest-%s", t.Name()),
		},
		Resources: config.Resources{
			CPU:    "4",
			Memory: "8Gi",
		},
	}
}
