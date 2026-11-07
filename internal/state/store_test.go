// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/i-zaitsev/pkg/assert"
)

func TestStore_Save(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	ws := &Workspace{ID: "test-1", Name: "test-ws", Status: "pending"}

	assert.ErrVal(t, store.Save(ws)).Nil().Fatal()
}

func TestStore_Load(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	ws := &Workspace{ID: "test-1", Name: "test-ws", Status: "pending"}
	assert.ErrVal(t, store.Save(ws)).Nil().Fatal()

	got, err := store.Load("test-1")
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, got.ID, ws.ID)
	assert.Eq(t, got.Name, ws.Name)
	assert.Eq(t, got.Status, ws.Status)
}

func TestStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	_, err := store.Load("nonexistent")
	assert.ErrVal(t, err).NotNil().Fatal()
}

func TestStore_List(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	list, err := store.List()
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, len(list), 0)

	assert.ErrVal(t, store.Save(&Workspace{ID: "ws-1", Name: "first"})).Nil().Fatal()
	assert.ErrVal(t, store.Save(&Workspace{ID: "ws-2", Name: "second"})).Nil().Fatal()

	list, err = store.List()
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, len(list), 2)
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	assert.ErrVal(t, store.Save(&Workspace{ID: "ws-1", Name: "test"})).Nil().Fatal()
	assert.ErrVal(t, store.Delete("ws-1")).Nil().Fatal()

	_, err := store.Load("ws-1")
	assert.ErrVal(t, err).NotNil().Fatal()
}

func TestStore_Delete_Idempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	assert.ErrVal(t, store.Delete("nonexistent")).Nil().Fatal()
}

func TestWorkspace_ExitStatus(t *testing.T) {
	code := func(n int) *int { return &n }

	tests := []struct {
		name string
		ws   Workspace
		want string
	}{
		{"nil exit code", Workspace{}, "pending"},
		{"nil with error", Workspace{ErrorMsg: "stopped by user"}, "stopped by user"},
		{"success", Workspace{ExitCode: code(0)}, "success"},
		{"non-zero", Workspace{ExitCode: code(1)}, "exit code 1"},
		{"non-zero with message", Workspace{ExitCode: code(137), ErrorMsg: "exit code 137"}, "exit code 137"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Eq(t, tt.ws.ExitStatus(), tt.want)
		})
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)
	n := 100

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Go(func() {
			assert.ErrVal(t, store.Save(&Workspace{
				ID:   fmt.Sprintf("ws-%d", i),
				Name: fmt.Sprintf("name-%d", i),
			})).Nil()
		})
	}

	wg.Wait()
	list, err := store.List()
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, len(list), n)
}

func TestStore_ParentID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	ws := &Workspace{
		ID:       "ws-2",
		Name:     "test",
		Status:   "pending",
		ParentID: "ws-1",
	}

	assert.ErrVal(t, store.Save(ws)).Nil().Fatal()
	got, err := store.Load("ws-2")
	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, got.ParentID, "ws-1")
}

func TestStore_Load_rejectsV1(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	v1 := `{"Version":1,"Workspaces":{"ws-1":{"ID":"ws-1","Name":"test","Status":"done"}}}`
	assert.ErrVal(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte(v1), 0o644)).Nil().Fatal()

	store := NewStore(dir)
	_, err := store.Load("ws-1")
	assert.ErrVal(t, err).NotNil().Fatal()
	assert.Eq(t, strings.Contains(err.Error(), "dwoe-migrate"), true)
}
