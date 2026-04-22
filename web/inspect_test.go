// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"testing"
	"time"

	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

func TestStartedFmt(t *testing.T) {
	ts := time.Date(2025, time.March, 15, 9, 30, 0, 0, time.UTC)
	tests := []struct {
		name string
		info workspaceInfo
		want string
	}{
		{"nil", workspaceInfo{}, "-"},
		{"non_nil", workspaceInfo{StartedAt: &ts}, "09:30 Mar 15 2025"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.StartedFmt(); got != tt.want {
				t.Errorf("StartedFmt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInspectWorkspace(t *testing.T) {
	serv, source := newTestServer()
	ws := workspace.New(state.EmptyWorkspace("test-id", "test-ws"), &config.Task{})
	ws.BasePath = "/tmp/ws-base"
	source.Set(ws.ID, ws)

	runHTTPTests(t, serv.handler, []httpCase{
		{"by_id", "/workspaces/inspect?q=test-id", 200, "test-id"},
		{"by_name", "/workspaces/inspect?q=test-ws", 200, "test-id"},
		{"paths", "/workspaces/inspect?q=test-id", 200, "/tmp/ws-base"},
		{"not_found", "/workspaces/inspect?q=none", 200, "not found"},
		{"missing_param", "/workspaces/inspect", 400, "required"},
	})
}
