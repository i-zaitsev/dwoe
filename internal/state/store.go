// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package state provides file-based persistence for workspace metadata.
package state

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/i-zaitsev/dwoe/internal/util"
)

// Workspace represents the persistent state of a single workspace.
// It tracks the workspace lifecycle from creation through completion or failure.
type Workspace struct {
	ID           string
	Name         string
	Status       string
	ExitCode     *int   `json:"exit_code,omitempty"`
	ErrorMsg     string `json:"error_msg,omitempty"`
	BasePath     string
	ContainerIDs map[string]string
	NetworkID    string
	CreatedAt    *time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

// EmptyWorkspace creates a placeholder structure with empty fields.
func EmptyWorkspace(id, name string) *Workspace {
	return &Workspace{
		ID:           id,
		Name:         name,
		ContainerIDs: make(map[string]string),
	}
}

func (ws *Workspace) ExitStatus() string {
	if ws.ExitCode == nil {
		if ws.ErrorMsg != "" {
			return ws.ErrorMsg
		}
		return "pending"
	}
	if *ws.ExitCode == 0 {
		return "success"
	}
	if ws.ErrorMsg != "" {
		return ws.ErrorMsg
	}
	return fmt.Sprintf("exit code %d", *ws.ExitCode)
}

// File is the on-disk JSON structure containing all workspace states.
type File struct {
	Version    int
	Workspaces map[string]*Workspace
}

// Store manages a persistent workspace state in a JSON file.
// File locking protects all operations for concurrent access.
type Store struct {
	path string
}

// NewStore creates a new Store that persists state to {dataDir}/state.json.
func NewStore(dataDir string) *Store {
	return &Store{path: filepath.Join(dataDir, "state.json")}
}

// Save creates or updates a workspace state.
func (s *Store) Save(ws *Workspace) error {
	slog.Debug("state: save", "id", ws.ID)
	return s.withLock(func() error {
		f, err := s.readFile()
		if err != nil {
			return err
		}
		f.Workspaces[ws.ID] = ws
		return s.writeFile(f)
	})
}

// Load retrieves a workspace state by ID. Returns an error if not found.
func (s *Store) Load(id string) (*Workspace, error) {
	slog.Debug("state: load", "id", id)
	var result *Workspace
	err := s.withLock(func() error {
		f, err := s.readFile()
		if err != nil {
			return err
		}
		ws, ok := f.Workspaces[id]
		if !ok {
			return &NotFoundError{ID: id}
		}
		result = ws
		return nil
	})
	return result, err
}

// List returns all workspace states.
func (s *Store) List() ([]*Workspace, error) {
	slog.Debug("state: list")
	var result []*Workspace
	err := s.withLock(func() error {
		f, err := s.readFile()
		if err != nil {
			return err
		}
		for _, ws := range f.Workspaces {
			result = append(result, ws)
		}
		return nil
	})
	return result, err
}

// Delete removes a workspace state by ID.
// No error if not found.
func (s *Store) Delete(id string) error {
	slog.Debug("state: delete", "id", id)
	return s.withLock(func() error {
		f, err := s.readFile()
		if err != nil {
			return err
		}
		delete(f.Workspaces, id)
		return s.writeFile(f)
	})
}

// UpdateStatus updates the status of a workspace.
// Returns an error if not found.
func (s *Store) UpdateStatus(id, status string) error {
	slog.Debug("state: update-status", "id", id, "status", status)
	return s.withLock(func() error {
		f, err := s.readFile()
		if err != nil {
			return err
		}
		ws, ok := f.Workspaces[id]
		if !ok {
			return &NotFoundError{ID: id}
		}
		ws.Status = status
		return s.writeFile(f)
	})
}

func (s *Store) readFile() (*File, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &File{Version: 1, Workspaces: map[string]*Workspace{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %s: %w", s.path, err)
	}
	var f File
	if errMarshal := json.Unmarshal(data, &f); errMarshal != nil {
		return nil, fmt.Errorf("failed to unmarshal file: %w", errMarshal)
	}
	if f.Workspaces == nil {
		f.Workspaces = map[string]*Workspace{}
	}
	return &f, nil
}

const empty = ""
const twoSpaces = "  "

func (s *Store) writeFile(f *File) error {
	data, err := json.MarshalIndent(f, empty, twoSpaces)
	if err != nil {
		return fmt.Errorf("cannot marshal state: %w", err)
	}
	return util.WriteFileAtomic(s.path, data, 0o644)
}

func (s *Store) withLock(fn func() error) error {
	f, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() {
		errUnlock := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		if errUnlock != nil {
			slog.Error("state: unlock", "path", s.path, "error", errUnlock)
		}
	}()
	return fn()
}
