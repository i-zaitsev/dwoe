// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"context"
	"embed"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	addr       string
	handler    http.Handler
	workspaces WorkspaceSource
}

type WorkspaceSource interface {
	Get(id string) (*workspace.Workspace, error)
	GetByName(name string) (*workspace.Workspace, error)
	List() ([]*workspace.Workspace, error)
	Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error)
	Diff(id string) (*workspace.DiffInfo, error)
	Sync(ctx context.Context, id string) error
	BatchRecords() ([]*batch.Record, error)
}

func NewServer(addr string) *Server {
	serv := Server{addr: addr}
	serv.handler = Routes(&serv)
	return &serv
}

func (s *Server) SetSource(ws WorkspaceSource) {
	s.workspaces = ws
}

func Routes(s *Server) http.Handler {
	mux := http.NewServeMux()
	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))
	mux.HandleFunc("GET /theme", s.setTheme)
	mux.HandleFunc("GET /", s.home)
	mux.HandleFunc("GET /batches", s.listBatches)
	mux.HandleFunc("GET /workspaces/diff", s.diffWorkspace)
	mux.HandleFunc("GET /workspaces/inspect", s.inspectWorkspace)
	mux.HandleFunc("GET /workspaces/list", s.listWorkspaces)
	mux.HandleFunc("GET /workspaces/logs", s.logsWorkspace)
	mux.HandleFunc("GET /workspaces/logs/connect", s.logsConnect)
	mux.HandleFunc("GET /workspaces/logs/stream", s.logsStream)
	mux.HandleFunc("GET /workspaces/logs/view", s.logsView)
	return mux
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	srv := &http.Server{Addr: s.addr, Handler: s.handler}

	go func() {
		<-ctx.Done()
		slog.Info("web: shutting down")
		if err := srv.Shutdown(context.Background()); err != nil {
			slog.Error("web: shutdown", "err", err)
		}
	}()

	slog.Info("web: listening", "addr", s.addr)

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
