// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"strings"
	"testing"

	"github.com/i-zaitsev/pkg/assert"
)

func TestParseLogEvents(t *testing.T) {
	longText := strings.Repeat("x", 200)
	tests := []struct {
		name       string
		line       string
		wantKinds  []string
		wantText   string
		wantTime   string
		wantDetail bool
	}{
		{
			"blank",
			"   ",
			nil, "", "", false,
		},
		{
			"container_plain",
			"[12:34:56] container started",
			[]string{"container"}, "container started", "12:34:56", false,
		},
		{
			"container_not_json",
			"just text",
			[]string{"container"}, "just text", "", false,
		},
		{
			"init",
			`{"type":"system","subtype":"init","timestamp":"2025-03-15T09:30:05Z","model":"claude-x","cwd":"/w","tools":["a","b"],"permissionMode":"default"}`,
			[]string{"init"}, "claude-x", "09:30:05", false,
		},
		{
			"task_started",
			`{"type":"system","subtype":"task_started","task_type":"explore","description":"look around"}`,
			[]string{"task_started"}, "explore: look around", "", false,
		},
		{
			"task_progress",
			`{"type":"system","subtype":"task_progress","last_tool_name":"Read","description":"reading","usage":{"total_tokens":10,"tool_uses":2}}`,
			[]string{"task_progress"}, "Read: reading (2 tool uses, 10 tokens)", "", false,
		},
		{
			"task_notification",
			`{"type":"system","subtype":"task_notification","status":"completed","summary":"all done"}`,
			[]string{"task_done"}, "completed: all done", "", false,
		},
		{
			"api_retry",
			`{"type":"system","subtype":"api_retry","attempt":1,"max_retries":3,"retry_delay_ms":500,"error_status":429,"error":"rate limited"}`,
			[]string{"api_retry"}, "attempt 1/3, retry in 500ms (status 429): rate limited", "", false,
		},
		{
			"rate_limit",
			`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"five_hour"}}`,
			[]string{"rate_limit"}, "allowed (five_hour)", "", false,
		},
		{
			"assistant_blocks",
			`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"},{"type":"thinking","thinking":"hmm"},{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`,
			[]string{"assistant", "thinking", "tool_use"}, "hello", "", false,
		},
		{
			"assistant_long_text_has_detail",
			`{"type":"assistant","message":{"content":[{"type":"text","text":"` + longText + `"}]}}`,
			[]string{"assistant"}, longText, "", true,
		},
		{
			"assistant_error",
			`{"type":"assistant","error":"overloaded","message":{"content":[]}}`,
			[]string{"error"}, "overloaded", "", false,
		},
		{
			"user_tool_result",
			`{"type":"user","message":{"content":[{"type":"tool_result","content":"line one\nline two"}]}}`,
			[]string{"tool_result"}, "line one", "", true,
		},
		{
			"user_tool_error",
			`{"type":"user","message":{"content":[{"type":"tool_result","is_error":true,"content":"failed"}]}}`,
			[]string{"error"}, "failed", "", false,
		},
		{
			"result",
			`{"type":"result","subtype":"success","num_turns":3,"duration_ms":125000,"total_cost_usd":0.5}`,
			[]string{"result"}, "success: 3 turns, 2m05s, $0.5000", "", false,
		},
		{
			"unknown",
			`{"type":"mystery","subtype":"thing"}`,
			[]string{"unknown"}, "mystery/thing", "", false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLogEvents(tt.line)

			var kinds []string
			for _, ev := range got {
				kinds = append(kinds, ev.Kind)
			}
			if len(kinds) != len(tt.wantKinds) {
				t.Fatalf("kinds = %v, want %v", kinds, tt.wantKinds)
			}
			for i, want := range tt.wantKinds {
				assert.Eq(t, kinds[i], want)
			}
			if len(got) == 0 {
				return
			}
			assert.Contains(t, got[0].Text, tt.wantText)
			assert.Eq(t, got[0].Time, tt.wantTime)
			assert.Eq(t, got[0].Detail != "", tt.wantDetail)
		})
	}
}

func TestParseLogEvents_ToolUseKeepsInput(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`

	got := parseLogEvents(line)

	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	assert.Eq(t, got[0].Text, "Bash: ls")
	assert.Eq(t, string(got[0].Raw), `{"command":"ls"}`)
}
