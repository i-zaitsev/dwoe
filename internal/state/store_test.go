// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package state

import (
	"fmt"
	"sync"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestStore_Save(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ws      *Workspace
		wantErr bool
	}{
		{
			name:    "save new workspace",
			ws:      &Workspace{ID: "test-1", Name: "test-ws", Status: "pending"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			store := NewStore(dir)
			err := store.Save(tt.ws)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_Load(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	ws := &Workspace{ID: "test-1", Name: "test-ws", Status: "pending"}
	assert.NotErr(t, store.Save(ws))

	got, err := store.Load("test-1")
	assert.NotErr(t, err)
	if got.ID != ws.ID || got.Name != ws.Name || got.Status != ws.Status {
		t.Errorf("Load() = %+v, want %+v", got, ws)
	}
}

func TestStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	_, err := store.Load("nonexistent")
	assert.Err(t, err)
}

func TestStore_List(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	list, err := store.List()
	assert.NotErr(t, err)
	assert.Equal(t, len(list), 0)

	assert.NotErr(t, store.Save(&Workspace{ID: "ws-1", Name: "first"}))
	assert.NotErr(t, store.Save(&Workspace{ID: "ws-2", Name: "second"}))

	list, err = store.List()
	assert.NotErr(t, err)
	assert.Equal(t, len(list), 2)
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	assert.NotErr(t, store.Save(&Workspace{ID: "ws-1", Name: "test"}))
	assert.NotErr(t, store.Delete("ws-1"))

	_, err := store.Load("ws-1")
	assert.Err(t, err)
}

func TestStore_Delete_Idempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)

	assert.NotErr(t, store.Delete("nonexistent"))
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
			assert.Equal(t, tt.ws.ExitStatus(), tt.want)
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
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			assert.NotErr(t, store.Save(&Workspace{ID: fmt.Sprintf("ws-%d", id), Name: fmt.Sprintf("name-%d", id)}))
		}(i)
	}
	wg.Wait()

	list, err := store.List()
	assert.NotErr(t, err)
	assert.Equal(t, len(list), n)
}
