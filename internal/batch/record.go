// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package batch manages batch work records and their entries.
package batch

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Record represents a batch of work.
// A record is a collection of tasks that are executed in parallel.
// All tasks are assumed to work on the same source code repository.
type Record struct {
	ID        string  `json:"id"`
	SourceDir string  `json:"source_dir"`
	CreatedAt string  `json:"created_at"`
	Entries   []Entry `json:"entries"`
}

// TotalTasks returns the total number of tasks in the batch.
func (r *Record) TotalTasks() int {
	return len(r.Entries)
}

// NewRecord creates a new batch record based on the provided tasks and workspace IDs.
func NewRecord(sourceDir string, taskFiles []string, ids []string) *Record {
	rec := &Record{
		ID:        uuid.New().String(),
		SourceDir: sourceDir,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Entries:   make([]Entry, len(taskFiles)),
	}
	for i, tf := range taskFiles {
		rec.Entries[i] = Entry{
			TaskFile:    tf,
			WorkspaceID: ids[i],
			Branch:      BranchName(tf),
		}
	}
	return rec
}

// Entry represents a single task in a batch of work.
// Each task works on a branch of a single workspace based on the provided task file.
// Generally, each task is independent and can be executed in parallel.
type Entry struct {
	TaskFile    string `json:"task_file"`
	WorkspaceID string `json:"workspace_id"`
	Branch      string `json:"branch"`
}

// BranchName returns a branch name based on the task file path.
// The name is derived from the task file's path.
func BranchName(taskFile string) string {
	dir := filepath.Dir(taskFile)
	base := strings.TrimSuffix(filepath.Base(taskFile), ".yaml")
	parts := strings.Split(filepath.ToSlash(dir), "/")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "/") + "/" + base
	}
	return base
}

// SaveRecord saves the batch work metadata to collect results when the tasks are done.
// When the record is saved, there is no protection from multiple writes.
// It is assumed that each batch is accessed by a single goroutine.
func SaveRecord(dataDir string, rec *Record) error {
	dir := filepath.Join(dataDir, "batches")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("batch save: %w", err)
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("batch save: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, rec.ID+".json"), data, 0o644)
}

// LoadRecord loads previously saved batch work metadata.
// Note that differently from the global lock file, batch metadata is not
// protected from concurrent access via swap files or locking.
func LoadRecord(dataDir, id string) (*Record, error) {
	return LoadRecordFromFile(filepath.Join(dataDir, "batches", id+".json"))
}

// LoadRecordFromFile loads previously saved batch work metadata from a file.
func LoadRecordFromFile(path string) (*Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("batch load: %w", err)
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("batch load: %w", err)
	}
	return &rec, nil
}

// LoadOrCreate tries to load a Record to add another related Entry.
// If the record does not exist, a new one is created.
func LoadOrCreate(dataDir, id, sourceDir string) (*Record, error) {
	rec, err := LoadRecord(dataDir, id)
	if err == nil {
		return rec, nil
	}
	if !os.IsNotExist(errors.Unwrap(err)) {
		return nil, err
	}
	return &Record{
		ID:        id,
		SourceDir: sourceDir,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func ReadRecords(dataDir, batchSubdir string) ([]*Record, error) {
	if batchSubdir == "" {
		batchSubdir = "batches"
	}

	batchesDir := filepath.Join(dataDir, batchSubdir)
	dir, err := os.ReadDir(batchesDir)
	if err != nil {
		return nil, fmt.Errorf("batch read: %w", err)
	}

	records := make([]*Record, 0, len(dir))
	for _, f := range dir {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		jsonPath := filepath.Join(batchesDir, f.Name())
		rec, errFile := LoadRecordFromFile(jsonPath)
		if errFile != nil {
			slog.Warn("failed to load batch record, skipping", "file", jsonPath, "error", errFile)
			continue
		}
		records = append(records, rec)
	}

	return records, nil
}
