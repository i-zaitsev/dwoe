// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"testing"
	"time"

	"github.com/i-zaitsev/pkg/assert"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

func TestShortID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"batch-abcdef12", "batch-ab"},
		{"ws-1", "ws-1"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Eq(t, shortID(tt.in), tt.want)
		})
	}
}

func TestFmtTime(t *testing.T) {
	ts := time.Date(2025, time.March, 15, 9, 30, 5, 0, time.UTC)

	assert.Eq(t, fmtTime(nil), "-")
	assert.Eq(t, fmtTime(&ts), "2025-03-15 09:30:05")
}

func TestDuration(t *testing.T) {
	start := time.Date(2025, 3, 15, 9, 0, 0, 0, time.UTC)
	after := func(d time.Duration) *time.Time {
		end := start.Add(d)
		return &end
	}
	tests := []struct {
		name string
		ws   data.Workspace
		want string
	}{
		{"not_started", data.Workspace{}, "not started"},
		{"running", data.Workspace{StartedAt: &start}, "running"},
		{"seconds", data.Workspace{StartedAt: &start, FinishedAt: after(5 * time.Second)}, "5.0s"},
		{"minutes", data.Workspace{StartedAt: &start, FinishedAt: after(2*time.Minute + 3*time.Second)}, "2m 3s"},
		{"hours", data.Workspace{StartedAt: &start, FinishedAt: after(time.Hour + time.Minute + time.Second)}, "1h 1m 1s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Eq(t, duration(tt.ws), tt.want)
		})
	}
}

func TestStatusClass(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"completed", "ok"},
		{"failed", "bad"},
		{"running", "run"},
		{"pending", "idle"},
		{"stopped", "idle"},
		{"whatever", "idle"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Eq(t, statusClass(tt.status), tt.want)
		})
	}
}

func TestEventClass(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"error", "err"},
		{"tool_use", "tool"},
		{"assistant", "ast"},
		{"thinking", "think"},
		{"result", "res"},
		{"container", "cont"},
		{"tool_result", "cont"},
		{"init", "sys"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			assert.Eq(t, eventClass(tt.kind), tt.want)
		})
	}
}

func TestDiffLines(t *testing.T) {
	got := diffLines("@@ -1 +1 @@\n+added\n-removed\n context\n")

	want := []diffLine{
		{"hunk", "@@ -1 +1 @@"},
		{"add", "+added"},
		{"del", "-removed"},
		{"", " context"},
	}
	if len(got) != len(want) {
		t.Fatalf("diffLines returned %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		assert.Eq(t, got[i], want[i])
	}
	assert.Eq(t, len(diffLines("")), 0)
}

func TestFuncMap_HasPathEscape(t *testing.T) {
	fn, ok := funcMap["pathEscape"].(func(string) string)
	assert.Eq(t, ok, true)

	tests := []struct {
		in   string
		want string
	}{
		{"ws-1", "ws-1"},
		{"a b", "a%20b"},
		{"a/b?c", "a%2Fb%3Fc"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Eq(t, fn(tt.in), tt.want)
		})
	}
}
