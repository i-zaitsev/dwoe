// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package data defines what the dashboard shows and how it reads it.
//
// The handlers depend only on the interfaces in this package. The host
// application implements them over its own storage.
package data

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrRecordNotFound is returned when a workspace or batch does not exist.
	ErrRecordNotFound = errors.New("record not found")

	// ErrUnavailable is returned when a record exists but part of it cannot
	// be read, for example the logs or the diff of a workspace.
	ErrUnavailable = errors.New("data unavailable")
)

// Models groups the models used by the dashboard.
type Models struct {
	Workspaces WorkspaceModel
	Batches    BatchModel
}

// WorkspaceModel reads workspaces.
type WorkspaceModel interface {
	// Get returns the current state of the workspace with the given ID or name.
	Get(ctx context.Context, ref string) (Workspace, error)
	List(f WorkspaceFilter) ([]Workspace, error)
	Diff(ref string) (Diff, error)
	// Logs calls emit for every log event of the workspace. With follow set
	// it keeps emitting until the workspace stops or ctx is done.
	Logs(ctx context.Context, ref string, follow bool, emit func(LogEvent) error) error
}

// BatchModel reads batches of workspaces.
type BatchModel interface {
	List() ([]Batch, error)
	// Get returns the batch with its per-workspace changes.
	Get(id string) (Batch, error)
}

// WorkspaceFilter narrows a workspace list. Empty fields match everything.
type WorkspaceFilter struct {
	Name  string
	Batch string
}

// Workspace is an isolated agent run.
type Workspace struct {
	ID          string
	Name        string
	Description string
	Status      string
	Exit        string
	Model       string
	BasePath    string
	WorkDir     string
	ParentID    string
	BatchID     string
	BatchColor  int
	CreatedAt   *time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	TaskConfig  string
	Prompt      string
}

// Running reports whether the workspace is still working.
func (w Workspace) Running() bool {
	return w.Status == "running"
}

// Commit is one commit of a diff.
type Commit struct {
	Hash    string
	Message string
}

// Diff is the change a workspace made.
type Diff struct {
	Commits []Commit
	Stat    string
	Patch   string
}

// Batch is a group of workspaces started together.
type Batch struct {
	ID         string
	Workers    int
	StartedAt  *time.Time
	FinishedAt *time.Time
	Changes    []BatchChange
}

// BatchChange is the diff of one workspace of a batch.
type BatchChange struct {
	Workspace Workspace
	Diff      Diff
}

// LogEvent is one entry of a workspace log.
// Text is a one-line summary, Detail holds the full text when it is longer,
// and Raw carries the original JSON payload when there is one.
type LogEvent struct {
	Time   string
	Kind   string
	Text   string
	Detail string
	Raw    []byte
}
