// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"bufio"
	"context"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

func TestRenderPrettyLogLine_Container(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		in    string
		wantT string
		wantB string
	}{
		{"with_bracket_time", "[08:32:15] Starting workspace", "08:32:15", "Starting workspace"},
		{"without_time", "Starting workspace", "", "Starting workspace"},
		{"empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderPrettyLogLine(tt.in)
			if tt.in == "" {
				assert.Equal(t, got, "")
				return
			}
			assert.Contains(t, got, `class="row cont"`)
			if tt.wantT != "" {
				assert.Contains(t, got, tt.wantT)
			}
			assert.Contains(t, got, tt.wantB)
		})
	}
}

func TestRenderPrettyLogLine_Assistant_Text(t *testing.T) {
	t.Parallel()
	in := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello world"}]}}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="row ast"`)
	assert.Contains(t, got, `ast·txt`)
	assert.Contains(t, got, "hello world")
}

func TestRenderPrettyLogLine_Assistant_Thinking_And_ToolUse(t *testing.T) {
	t.Parallel()
	in := `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"thinking","thinking":"pondering\nnext step"},` +
		`{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls /tmp"}}` +
		`]}}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="expand think"`)
	assert.Contains(t, got, "pondering")
	assert.Contains(t, got, `class="expand tool"`)
	assert.Contains(t, got, "Bash")
	assert.Contains(t, got, "ls /tmp")
}

func TestRenderPrettyLogLine_User_ToolResult_Error(t *testing.T) {
	t.Parallel()
	in := `{"type":"user","message":{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"t1","is_error":true,"content":"Exit 1\nbad thing"}` +
		`]}}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="expand err"`)
	assert.Contains(t, got, "Exit 1")
}

func TestRenderPrettyLogLine_User_ToolResult_Ok(t *testing.T) {
	t.Parallel()
	in := `{"type":"user","message":{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"t1","content":"file1\nfile2"}` +
		`]}}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="expand stdout"`)
	assert.Contains(t, got, "file1")
}

func TestRenderPrettyLogLine_Result(t *testing.T) {
	t.Parallel()
	in := `{"type":"result","subtype":"success","is_error":false,"duration_ms":505000,"num_turns":42,"total_cost_usd":2.18,"usage":{"input_tokens":100,"output_tokens":50}}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="expand res"`)
	assert.Contains(t, got, "$2.18")
	assert.Contains(t, got, "8m 25s")
	assert.Contains(t, got, "42 turns")
}

func TestRenderPrettyLogLine_SystemInit(t *testing.T) {
	t.Parallel()
	in := `{"type":"system","subtype":"init","session_id":"abc","model":"claude-opus-4-6","cwd":"/workspace","tools":["Bash","Read"],"permissionMode":"default"}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="expand sys"`)
	assert.Contains(t, got, "abc")
	assert.Contains(t, got, "claude-opus-4-6")
	assert.Contains(t, got, "tools=</span>2")
}

func TestRenderPrettyLogLine_RateLimit(t *testing.T) {
	t.Parallel()
	in := `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"five_hour"}}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="row rate"`)
	assert.Contains(t, got, "allowed")
}

func TestRenderPrettyLogLine_Unknown(t *testing.T) {
	t.Parallel()
	in := `{"type":"something_new","foo":"bar"}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="expand sys"`)
	assert.Contains(t, got, "something_new")
}

func TestRenderPrettyLogLine_InvalidJSON(t *testing.T) {
	t.Parallel()
	got := renderPrettyLogLine("not json at all")
	assert.Contains(t, got, `class="row cont"`)
	assert.Contains(t, got, "not json at all")
}

func TestLogsView_PrettyFlag(t *testing.T) {
	serv, source := newTestServer()
	ws := workspace.New(state.EmptyWorkspace("ws-1", "test-ws"), &config.Task{})
	source.Set(ws.ID, ws)

	runHTTPTests(t, serv.handler, []httpCase{
		{"tape_page", "/workspaces/logs/view?q=ws-1&pretty=1", 200, `class="tape"`},
		{"tape_rows", "/workspaces/logs/view?q=ws-1&pretty=1", 200, `id="rows"`},
		{"tape_back_link", "/workspaces/logs/view?q=ws-1&pretty=1", 200, `Simple view`},
		{"classic_still_works", "/workspaces/logs/view?q=ws-1", 200, `Tape view`},
	})
}

func TestRenderPrettyLogLine_StdoutLongSingleLine(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("abcdefghij", 50) // 500 chars, no newline
	in := `{"type":"user","message":{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"t1","content":` +
		`"` + long + `"` +
		`}]}}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="expand stdout"`)
	assert.Contains(t, got, long)
}

func TestRenderPrettyLogLine_AssistantLongText(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("word ", 40) // 200 chars, no newline
	in := `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"text","text":"` + strings.TrimRight(long, " ") + `"}` +
		`]}}`
	got := renderPrettyLogLine(in)
	assert.Contains(t, got, `class="expand ast"`)
}

func TestRenderPrettyLogLine_SampleJSONL(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "schema", "testdata", "sample.jsonl")
	f, err := os.Open(path)
	assert.NotErr(t, err)
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	lines := 0
	for sc.Scan() {
		got := renderPrettyLogLine(sc.Text())
		assert.NotZero(t, len(got))
		lines++
	}
	assert.NotErr(t, sc.Err())
	assert.NotZero(t, lines)
}

func TestNewLineMatcher(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"empty_matches_everything", "", "anything", true},
		{"substring_hit", "foo", "say foo bar", true},
		{"substring_miss", "foo", "bar baz", false},
		{"case_insensitive", "PYTHON", "python main", true},
		{"regex_dot_star", ".*\\.py$", "files main.py", true},
		{"regex_anchor_miss", "^foo", "the foo", false},
		{"invalid_regex_falls_back_to_literal", "*.py", "a *.py here", true},
		{"invalid_regex_fallback_miss", "*.py", "just main.py", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newLineMatcher(tt.pattern)
			assert.Equal(t, m(tt.input), tt.want)
		})
	}
}

func TestLogsWorkspace_ServerSideFilter(t *testing.T) {
	serv, source := newTestServer()
	ws := workspace.New(state.EmptyWorkspace("ws-1", "test-ws"), &config.Task{})
	source.Set(ws.ID, ws)
	source.logsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		content := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"keep apples"}]}}` + "\n" +
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"drop bananas"}]}}` + "\n"
		return io.NopCloser(strings.NewReader(content)), nil
	}

	t.Run("match_keeps_only_matching", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/workspaces/logs?q=ws-1&pretty=1&filter=apples", nil)
		serv.handler.ServeHTTP(rr, req)
		body := rr.Body.String()
		assert.Equal(t, rr.Code, 200)
		assert.Contains(t, body, "keep apples")
		if strings.Contains(body, "drop bananas") {
			t.Errorf("expected non-matching row to be filtered out, got: %s", body)
		}
	})

	t.Run("empty_filter_shows_all", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/workspaces/logs?q=ws-1&pretty=1&filter=", nil)
		serv.handler.ServeHTTP(rr, req)
		body := rr.Body.String()
		assert.Contains(t, body, "keep apples")
		assert.Contains(t, body, "drop bananas")
	})
}

func TestLogsWorkspace_PrettyFragment(t *testing.T) {
	serv, source := newTestServer()
	ws := workspace.New(state.EmptyWorkspace("ws-1", "test-ws"), &config.Task{})
	source.Set(ws.ID, ws)
	source.logsFn = func(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
		content := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi there"}]}}` + "\n"
		return io.NopCloser(strings.NewReader(content)), nil
	}

	runHTTPTests(t, serv.handler, []httpCase{
		{"fragment", "/workspaces/logs?q=ws-1&pretty=1", 200, `class="row ast"`},
		{"fragment_has_text", "/workspaces/logs?q=ws-1&pretty=1", 200, "hi there"},
	})
}
