// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

var addr = "127.0.0.1:8080"

type fakeSource struct {
	data           map[string]*workspace.Workspace
	logsFn         func(ctx context.Context, id string, follow bool) (io.ReadCloser, error)
	diffFn         func(id string) (*workspace.DiffInfo, error)
	batchRecordsFn func() ([]*batch.Record, error)
}

func newFakeSource() *fakeSource {
	return &fakeSource{
		data: make(map[string]*workspace.Workspace),
	}
}

func (f *fakeSource) Set(id string, ws *workspace.Workspace) {
	f.data[id] = ws
}

func (f *fakeSource) Get(id string) (*workspace.Workspace, error) {
	if ws, ok := f.data[id]; ok {
		return ws, nil
	}
	return nil, &state.NotFoundError{ID: id}
}

func (f *fakeSource) GetByName(name string) (*workspace.Workspace, error) {
	for _, ws := range f.data {
		if ws.Name == name {
			return ws, nil
		}
	}
	return nil, &state.NotFoundError{ID: name}
}

func (f *fakeSource) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	if f.logsFn != nil {
		return f.logsFn(ctx, id, follow)
	}
	return nil, &state.NotFoundError{ID: id}
}

func (f *fakeSource) List() ([]*workspace.Workspace, error) {
	var list []*workspace.Workspace
	for _, ws := range f.data {
		list = append(list, ws)
	}
	return list, nil
}

func (f *fakeSource) Diff(id string) (*workspace.DiffInfo, error) {
	if f.diffFn != nil {
		return f.diffFn(id)
	}
	return nil, &state.NotFoundError{ID: id}
}

func (f *fakeSource) Sync(_ context.Context, _ string) error {
	return nil
}

func (f *fakeSource) BatchRecords() ([]*batch.Record, error) {
	if f.batchRecordsFn != nil {
		return f.batchRecordsFn()
	}
	return nil, nil
}

func newTestServer() (*Server, *fakeSource) {
	serv := NewServer(addr)
	source := newFakeSource()
	serv.workspaces = source
	return serv, source
}

type httpCase struct {
	name        string
	url         string
	wantStatus  int
	wantContain string
}

func runHTTPTests(t *testing.T, handler http.Handler, tests []httpCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.url, nil)
			handler.ServeHTTP(rr, req)

			assert.Equal(t, rr.Code, tt.wantStatus)
			assert.Contains(t, rr.Body.String(), tt.wantContain)
		})
	}
}
