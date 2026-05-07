package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

type exitError struct {
	code int
}

var errRunInterrupted = errors.New("interrupted")

func (e *exitError) Error() string {
	return fmt.Sprintf("workspace failed with exit code %d", e.code)
}

func runAttached(e *cli.Env, manager *workspace.Manager, id, name string) error {
	ctx := e.Context()

	logs, err := manager.Logs(ctx, id, true)
	if err != nil {
		slog.Error("run: cannot read running job logs", "err", err)
		errStop := manager.Stop(context.Background(), id, time.Minute)
		e.Error("failed to read the logs from attached worker")
		if errStop != nil {
			return fmt.Errorf("fatal: cannot stop the workspace: %w", errStop)
		}
		return err
	}

	lines := make(chan string)
	logCtx, logCancel := context.WithCancel(ctx)
	go cli.ScanLogs(logCtx, logs, lines)

	for line := range lines {
		e.Print("%s\n", line)
	}

	exitCode, waitErr := manager.Wait(ctx, id)
	logCancel()

	if ctx.Err() != nil {
		e.Print("\nInterrupted. Stopping workspace %s...\n", name)
		bgCtx := context.Background()
		if errStop := manager.Stop(bgCtx, id, 30*time.Second); errStop != nil {
			slog.Error("run: stop on interrupt", "err", errStop)
		}
		if errCleanup := manager.Cleanup(bgCtx, id); errCleanup != nil {
			slog.Error("run: cleanup on interrupt", "err", errCleanup)
		}
		return errRunInterrupted
	}

	status := workspace.StatusCompleted
	if waitErr != nil || exitCode != 0 {
		status = workspace.StatusFailed
	}
	if errCleanup := manager.Cleanup(context.Background(), id); errCleanup != nil {
		slog.Error("run: cleanup", "err", errCleanup)
	}

	e.Print("Workspace %s: %s (exit code %d)\n", name, status, exitCode)
	if exitCode != 0 {
		return &exitError{code: exitCode}
	}
	return waitErr
}
