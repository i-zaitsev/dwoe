// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package log

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func testTime() time.Time {
	return time.Date(2000, 1, 1, 12, 34, 56, 999, time.UTC)
}

func TestPrettyHandler_MessageFormatting(t *testing.T) {
	testCases := []struct {
		level    slog.Level
		msg      string
		expected string
	}{
		{
			slog.LevelDebug,
			"test",
			"[2000-01-01 12:34:56][DBG] test",
		},
		{
			slog.LevelInfo,
			"",
			"[2000-01-01 12:34:56][INF]",
		},
		{
			slog.LevelInfo,
			"info message",
			"[2000-01-01 12:34:56][INF] info message",
		},
	}

	for _, tc := range testCases {
		var buf strings.Builder
		h := NewPrettyHandler(&buf, tc.level, "/tmp/test")
		r := slog.NewRecord(testTime(), tc.level, tc.msg, 0)

		err := h.Handle(context.Background(), r)
		log := strings.TrimSpace(buf.String())

		assert.NotErr(t, err)
		assert.NotEqual(t, log, "")
		assert.Equal(t, log, tc.expected)
	}
}

func TestPrettyHandler_SourceLogForLevels(t *testing.T) {
	oldFrame := getFrame
	getFrame = func(pc uintptr) runtime.Frame {
		return runtime.Frame{
			PC:       pc,
			Func:     nil,
			Function: "test()",
			File:     "/tmp/test/pkg/logic/mod.go",
			Line:     123,
			Entry:    0,
		}
	}
	t.Cleanup(func() {
		getFrame = oldFrame
	})

	testCases := []struct {
		level    slog.Level
		expected string
	}{
		{
			slog.LevelDebug,
			"[2000-01-01 12:34:56][DBG] test",
		},
		{
			slog.LevelInfo,
			"[2000-01-01 12:34:56][INF] test",
		},
		{
			slog.LevelWarn,
			"[2000-01-01 12:34:56][WRN] test (/pkg/logic/mod.go:123)",
		},
		{
			slog.LevelError,
			"[2000-01-01 12:34:56][ERR] test (/pkg/logic/mod.go:123)",
		},
	}

	for _, tc := range testCases {
		var buf strings.Builder
		h := NewPrettyHandler(&buf, tc.level, "/tmp/test")
		r := slog.NewRecord(testTime(), tc.level, "test", 1)

		err := h.Handle(context.Background(), r)

		assert.NotErr(t, err)
		assert.Equal(t, strings.TrimSpace(buf.String()), tc.expected)
	}
}
