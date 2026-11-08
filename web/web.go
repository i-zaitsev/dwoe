// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package web connects the dashboard of package hio to dwoe: it implements the
// dashboard models over the workspace manager and runs the server.
package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/i-zaitsev/dwoe/internal/config/provider"
	"github.com/i-zaitsev/dwoe/internal/workspace"
	"github.com/i-zaitsev/dwoe/pkg/hio"
	"github.com/i-zaitsev/dwoe/pkg/hio/config"
	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

// Main runs the standalone dashboard and returns the process exit code.
func Main(args []string) int {
	cfg, err := config.Parse(args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}
	env := config.NewEnv(cfg)

	provider.RegisterDefaultProviders()

	models, err := Open(env.DataDir())
	if err != nil {
		env.AppLog().Error(err.Error())
		return 1
	}

	if err = Serve(context.Background(), env, models); err != nil {
		env.AppLog().Error(err.Error())
		return 1
	}

	return 0
}

// NewModels returns the dashboard models over a workspace manager.
func NewModels(mgr *workspace.Manager) data.Models {
	return data.Models{
		Workspaces: &workspaceModel{mgr: mgr},
		Batches:    &batchModel{mgr: mgr},
	}
}

// Open creates a workspace manager for datadir, syncs it, and returns models over it.
func Open(datadir string) (data.Models, error) {
	mgr, err := workspace.NewManager(datadir)
	if err != nil {
		return data.Models{}, err
	}
	if err = mgr.SyncAll(context.Background()); err != nil {
		return data.Models{}, fmt.Errorf("sync workspaces: %w", err)
	}
	return NewModels(mgr), nil
}

// Serve runs the dashboard until the server fails or ctx is done.
func Serve(ctx context.Context, env *config.Env, models data.Models) error {
	app, err := hio.NewApp(env, models)
	if err != nil {
		return err
	}

	srv := hio.NewServer(env, app.Routes())
	stop := context.AfterFunc(ctx, func() { _ = srv.Close() })
	defer stop()

	env.AppLog().Info("starting server", "addr", srv.Addr, "log_level", env.LogLevel())

	if err = srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
