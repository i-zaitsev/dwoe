// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// Env is a command execution environment.
//
// Each command should print its user-facing output and errors via Env and avoid
// printing directly to ensure that the commands are decoupled from I/O ops.
// Additionally, the Env provides a method to construct a workspace manager, which
// might be a different implementation depending on the context. (E.g., testing.)
//
// The use of ctx is up to the commands. The main function injects this value, which
// might enable interruptions and timeouts. By default, a simple [context.Background]
// is used.
type Env struct {
	stdout     io.Writer
	stderr     io.Writer
	ctx        context.Context
	dataDir    string
	noProxy    bool
	newManager func() (*workspace.Manager, error)
}

// NewEnv creates an Env that writes to the given stdout and stderr writers.
func NewEnv(wOut, wErr io.Writer) *Env {
	return &Env{
		stdout: wOut,
		stderr: wErr,
		ctx:    context.Background(),
	}
}

// Stdout returns the standard output writer.
func (e *Env) Stdout() io.Writer {
	return e.stdout
}

// Stderr returns the standard error writer.
func (e *Env) Stderr() io.Writer {
	return e.stderr
}

// DataDir returns the configured data directory path.
func (e *Env) DataDir() string {
	return e.dataDir
}

// Manager returns a new instance.
// The created instance is immediately used to sync the state of workspaces.
// If creation or syncing fails, the error is returned.
func (e *Env) Manager() (*workspace.Manager, error) {
	m, errNew := e.newManager()
	if errNew != nil {
		return nil, errNew
	}
	if errSync := m.SyncAll(e.ctx); errSync != nil {
		slog.Warn("env: sync-all", "err", errSync)
		return nil, errSync
	}
	return m, nil
}

// Context returns the environment's context.
func (e *Env) Context() context.Context {
	return e.ctx
}

// Print writes formatted output to stdout.
func (e *Env) Print(format string, args ...any) {
	_, _ = fmt.Fprintf(e.stdout, format, args...)
}

// Error writes formatted output to stderr.
func (e *Env) Error(format string, args ...any) {
	_, _ = fmt.Fprintf(e.stderr, format, args...)
}

// SetContext replaces the environment's context.
func (e *Env) SetContext(ctx context.Context) {
	e.ctx = ctx
}

// SetDataDir sets the data directory path.
func (e *Env) SetDataDir(dir string) {
	e.dataDir = dir
}

// NoProxy reports whether the proxy container is disabled.
func (e *Env) NoProxy() bool { return e.noProxy }

// SetNoProxy enables or disables the proxy container.
func (e *Env) SetNoProxy(v bool) { e.noProxy = v }

// SetNewManager replaces the factory function used to create a workspace manager.
func (e *Env) SetNewManager(fn func() (*workspace.Manager, error)) {
	e.newManager = fn
}
