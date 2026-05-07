package commands

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestRefineCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		args       []string
		wantParent string
		wantDo     string
		wantWork   string
		wantName   string
		wantDetach bool
	}{
		{
			name:       "do_long",
			args:       []string{"--do", "extend it", "parent-ws"},
			wantParent: "parent-ws",
			wantDo:     "extend it",
		},
		{
			name:       "do_short",
			args:       []string{"-d", "extend it", "parent-ws"},
			wantParent: "parent-ws",
			wantDo:     "extend it",
		},
		{
			name:       "work_long",
			args:       []string{"--work", "task.md", "parent-ws"},
			wantParent: "parent-ws",
			wantWork:   "task.md",
		},
		{
			name:       "work_short",
			args:       []string{"-w", "task.md", "parent-ws"},
			wantParent: "parent-ws",
			wantWork:   "task.md",
		},
		{
			name:       "detach_long",
			args:       []string{"--detach", "--do", "x", "parent-ws"},
			wantParent: "parent-ws",
			wantDo:     "x",
			wantDetach: true,
		},
		{
			name:       "detach_short",
			args:       []string{"-D", "--do", "x", "parent-ws"},
			wantParent: "parent-ws",
			wantDo:     "x",
			wantDetach: true,
		},
		{
			name:       "with_name",
			args:       []string{"--name", "child", "--do", "x", "parent-ws"},
			wantParent: "parent-ws",
			wantDo:     "x",
			wantName:   "child",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := new(cmdRefine)
			assert.NotErr(t, cmd.Parse(tc.args))
			assert.Equal(t, cmd.parentNameOrID, tc.wantParent)
			assert.Equal(t, cmd.do, tc.wantDo)
			assert.Equal(t, cmd.work, tc.wantWork)
			assert.Equal(t, cmd.name, tc.wantName)
			assert.Equal(t, cmd.detach, tc.wantDetach)
		})
	}
}

func TestRefineCmd_Parse_MissingParent(t *testing.T) {
	t.Parallel()
	cmd := new(cmdRefine)
	err := cmd.Parse([]string{"--do", "x"})
	assert.ErrAs[*cli.ArgMissingError](t, err)
}

func TestRefineCmd_Parse_MissingDoAndWork(t *testing.T) {
	t.Parallel()
	cmd := new(cmdRefine)
	err := cmd.Parse([]string{"parent-ws"})
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "either --do or --work is required")
}

func TestRefineCmd_Parse_DoAndWorkConflict(t *testing.T) {
	t.Parallel()
	cmd := new(cmdRefine)
	err := cmd.Parse([]string{"--do", "x", "--work", "f.md", "parent-ws"})
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "cannot use both --do and --work")
}

func TestRefineCmd_Parse_UnknownFlag(t *testing.T) {
	t.Parallel()
	cmd := new(cmdRefine)
	err := cmd.Parse([]string{"--bogus", "parent-ws"})
	assert.ErrAs[*cli.FlagParseError](t, err)
}

func createCompletedParent(t *testing.T, setup *cmdTestSetup, id string) {
	t.Helper()
	dir := t.TempDir()
	ws := createWorkspace(t, dir, id, "completed", refineNow())
	if err := os.MkdirAll(filepath.Join(ws.BasePath, "workspace"), 0o755); err != nil {
		t.Fatal(err)
	}
	setup.state.Data[id] = ws
}

func refineNow() *time.Time {
	t := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	return &t
}

func TestRefineCmd_Run_RejectsActive(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"running", "pending"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			setup := newCmdTestSetup(t)
			setup.state.Data["ws-1"] = createWorkspace(t, t.TempDir(), "ws-1", status, refineNow())
			cmd := &cmdRefine{parentNameOrID: "ws-1", do: "extend it"}

			err := cmd.Run(setup.env)

			assert.Err(t, err)
			assert.Contains(t, err.Error(), "still "+status)
		})
	}
}

func TestRefineCmd_Run_Detach(t *testing.T) {
	t.Parallel()

	t.Run("do", func(t *testing.T) {
		t.Parallel()
		setup := newCmdTestSetup(t)
		createCompletedParent(t, setup, "ws-done")
		cmd := &cmdRefine{parentNameOrID: "ws-done", do: "extend it", detach: true}

		err := cmd.Run(setup.env)

		assert.NotErr(t, err)
		assert.ContainsAll(t, setup.stdout.String(),
			"Refining workspace:",
			"Parent:",
			"Status: running",
			"View logs:",
		)
	})

	t.Run("work", func(t *testing.T) {
		t.Parallel()
		setup := newCmdTestSetup(t)
		createCompletedParent(t, setup, "ws-done")
		workFile := filepath.Join(t.TempDir(), "prompt.md")
		testutil.WriteFile(t, workFile, "do the thing from file")
		cmd := &cmdRefine{parentNameOrID: "ws-done", work: workFile, detach: true}

		err := cmd.Run(setup.env)

		assert.NotErr(t, err)
		assert.ContainsAll(t, setup.stdout.String(), "Refining workspace:")
	})

	t.Run("creates_new_state_entry", func(t *testing.T) {
		t.Parallel()
		setup := newCmdTestSetup(t)
		createCompletedParent(t, setup, "ws-done")
		before := len(setup.state.Data)
		cmd := &cmdRefine{parentNameOrID: "ws-done", do: "extend it", detach: true}

		err := cmd.Run(setup.env)

		assert.NotErr(t, err)
		assert.Equal(t, len(setup.state.Data), before+1)
		assert.Equal(t, setup.state.Data["ws-done"].Status, "completed")
	})
}

func TestRefineCmd_Run_Attached(t *testing.T) {
	t.Parallel()
	setup := newCmdTestSetup(t)
	setup.docker.ContainerLogsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		return logReader("log-line-1\n<promise>DONE</promise>\n"), nil
	}
	setup.docker.WaitContainerFn = func(_ context.Context, _ string) (int, error) {
		return 0, nil
	}
	createCompletedParent(t, setup, "ws-done")
	cmd := &cmdRefine{parentNameOrID: "ws-done", do: "extend it"}

	err := cmd.Run(setup.env)

	assert.NotErr(t, err)
	assert.ContainsAll(t, setup.stdout.String(), "log-line-1", "completed")
}
