// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package schema

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func readLines(t *testing.T, path string) [][]byte {
	t.Helper()
	f, err := os.Open(path)
	assert.NotErr(t, err)
	defer f.Close()

	var out [][]byte
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := make([]byte, len(sc.Bytes()))
		copy(line, sc.Bytes())
		out = append(out, line)
	}
	assert.NotErr(t, sc.Err())
	return out
}

func TestParse_Samples(t *testing.T) {
	t.Parallel()
	lines := readLines(t, filepath.Join("testdata", "sample.jsonl"))

	tests := []struct {
		wantType    string
		wantSubtype string
		wantGoType  string
	}{
		{"system", "init", "*schema.SystemInit"},
		{"assistant", "", "*schema.Assistant"},
		{"assistant", "", "*schema.Assistant"},
		{"assistant", "", "*schema.Assistant"},
		{"user", "", "*schema.User"},
		{"user", "", "*schema.User"},
		{"system", "task_started", "*schema.TaskStarted"},
		{"system", "task_progress", "*schema.TaskProgress"},
		{"system", "task_notification", "*schema.TaskNotification"},
		{"rate_limit_event", "", "*schema.RateLimitEvent"},
		{"result", "success", "*schema.Result"},
		{"system", "api_retry", "*schema.APIRetry"},
	}
	assert.Equal(t, len(lines), len(tests))

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%s_%s", i, tt.wantType, tt.wantSubtype), func(t *testing.T) {
			t.Parallel()
			v, err := Parse(lines[i])
			assert.NotErr(t, err)
			assert.Equal(t, fmt.Sprintf("%T", v), tt.wantGoType)

			env := envelopeOf(t, v)
			assert.Equal(t, env.Type, tt.wantType)
			assert.Equal(t, env.Subtype, tt.wantSubtype)
			assert.NotZero(t, len(env.Raw))
		})
	}
}

func TestParse_Faulty(t *testing.T) {
	t.Parallel()
	lines := readLines(t, filepath.Join("testdata", "faulty.jsonl"))

	tests := []struct {
		name    string
		wantErr error
		wantGo  string
		wantTop string
	}{
		{"empty", ErrEmpty, "", ""},
		{"whitespace_only", ErrEmpty, "", ""},
		{"plain_text", nil, "", ""},
		{"invalid_json", nil, "", ""},
		{"no_type", nil, "*schema.Unknown", "foo"},
		{"unknown_type", nil, "*schema.Unknown", "x"},
		{"unknown_subtype", nil, "*schema.Unknown", ""},
	}
	assert.Equal(t, len(lines), len(tests))

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := Parse(lines[i])
			switch tt.name {
			case "empty", "whitespace_only":
				assert.ErrIs(t, err, tt.wantErr)
				return
			case "plain_text", "invalid_json":
				assert.Err(t, err)
				return
			}
			assert.NotErr(t, err)
			assert.Equal(t, fmt.Sprintf("%T", v), tt.wantGo)
			u, ok := v.(*Unknown)
			assert.Condition(t, ok)
			if tt.wantTop != "" {
				assert.HasKey(t, u.Top, tt.wantTop)
			}
		})
	}
}

func TestParse_SystemInit_Spot(t *testing.T) {
	t.Parallel()
	lines := readLines(t, filepath.Join("testdata", "sample.jsonl"))
	v, err := Parse(lines[0])
	assert.NotErr(t, err)

	init, ok := v.(*SystemInit)
	assert.Condition(t, ok)
	assert.NotZero(t, init.Model)
	if !slices.Contains(init.Tools, "Bash") {
		t.Errorf("Tools missing Bash: %v", init.Tools)
	}
	assert.NotZero(t, init.CWD)
}

func TestParse_Assistant_ToolUse_Spot(t *testing.T) {
	t.Parallel()
	lines := readLines(t, filepath.Join("testdata", "sample.jsonl"))
	// Line 3 (index 2) is the assistant/tool_use fixture.
	v, err := Parse(lines[2])
	assert.NotErr(t, err)

	a, ok := v.(*Assistant)
	assert.Condition(t, ok)
	assert.Equal(t, len(a.Message.Content), 1)
	block := a.Message.Content[0]
	assert.Equal(t, block.Type, "tool_use")
	assert.NotZero(t, block.Name)
	assert.NotZero(t, len(block.Input))
}

func TestParse_Raw_PreservesUnmodelled(t *testing.T) {
	t.Parallel()
	lines := readLines(t, filepath.Join("testdata", "sample.jsonl"))
	v, err := Parse(lines[0])
	assert.NotErr(t, err)

	init, ok := v.(*SystemInit)
	assert.Condition(t, ok)
	var top map[string]json.RawMessage
	assert.NotErr(t, json.Unmarshal(init.Raw, &top))
	// output_style is emitted by the CLI but intentionally not modelled
	// on SystemInit; renderers reach it via Raw.
	assert.HasKey(t, top, "output_style")
}

// envelopeOf returns the Envelope embedded in any concrete schema type.
// Test-only helper kept here to avoid exposing it from the package.
func envelopeOf(t *testing.T, v any) *Envelope {
	t.Helper()
	type envHolder interface{ envelope() *Envelope }
	h, ok := v.(envHolder)
	if !ok {
		t.Fatalf("value %T does not embed Envelope", v)
	}
	return h.envelope()
}
