package migrate

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/state"
)

func TestState_migratesV1(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/state_v1.json")
	assert.NotErr(t, err)

	out, count, err := State(data)
	assert.NotErr(t, err)
	assert.Equal(t, count, 1)

	var f state.File
	assert.NotErr(t, json.Unmarshal(out, &f))
	assert.Equal(t, f.Version, 2)

	ws := f.Workspaces["ws-abc-123"]
	assert.NotNil(t, ws)
	assert.Equal(t, ws.ID, "ws-abc-123")
	assert.Equal(t, ws.Name, "lucky-mighty-squid")
	assert.Equal(t, ws.Status, "completed")
	assert.Equal(t, ws.BasePath, "/tmp/workspaces/ws-abc-123")
	assert.NotNil(t, ws.ExitCode)
	assert.Equal(t, *ws.ExitCode, 0)
	assert.Equal(t, ws.ParentID, "ws-parent-1")
	assert.Equal(t, ws.NetworkID, "net-123")
	assert.HasKey(t, ws.ContainerIDs, "agent")
	assert.HasKey(t, ws.ContainerIDs, "proxy")
	assert.NotNil(t, ws.CreatedAt)
	assert.NotNil(t, ws.StartedAt)
	assert.NotNil(t, ws.FinishedAt)

	ws2 := f.Workspaces["ws-def-456"]
	assert.NotNil(t, ws2)
	assert.Equal(t, ws2.ErrorMsg, "timeout")
	assert.Nil(t, ws2.ExitCode)
}

func TestState_noopOnV2(t *testing.T) {
	t.Parallel()

	v2 := `{"version":2,"workspaces":{"ws-1":{"id":"ws-1","name":"test","status":"done","base_path":"/tmp"}}}`
	out, count, err := State([]byte(v2))
	assert.NotErr(t, err)
	assert.Equal(t, count, 0)
	assert.Equal(t, string(out), v2)
}
