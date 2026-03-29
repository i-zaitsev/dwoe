// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

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
