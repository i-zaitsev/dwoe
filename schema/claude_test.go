// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package schema

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/i-zaitsev/dwoe/schema/schematest"
	"github.com/i-zaitsev/pkg/assert"
)

func TestParse_Samples(t *testing.T) {
	t.Parallel()
	lines := schematest.Lines(t, "sample.jsonl")

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
	assert.Eq(t, len(lines), len(tests))

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%s_%s", i, tt.wantType, tt.wantSubtype), func(t *testing.T) {
			t.Parallel()
			v, err := Parse([]byte(lines[i]))
			assert.ErrVal(t, err).Nil().Fatal()
			assert.Eq(t, fmt.Sprintf("%T", v), tt.wantGoType)

			env := envelopeOf(t, v)
			assert.Eq(t, env.Type, tt.wantType)
			assert.Eq(t, env.Subtype, tt.wantSubtype)
			assert.NotZero(t, len(env.Raw))
		})
	}
}

func TestParse_Faulty(t *testing.T) {
	t.Parallel()
	lines := schematest.Lines(t, "faulty.jsonl")

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
	assert.Eq(t, len(lines), len(tests))

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := Parse([]byte(lines[i]))
			switch tt.name {
			case "empty", "whitespace_only":
				assert.ErrVal(t, err).Is(tt.wantErr).Fatal()
				return
			case "plain_text", "invalid_json":
				assert.ErrVal(t, err).NotNil().Fatal()
				return
			}
			assert.ErrVal(t, err).Nil().Fatal()
			assert.Eq(t, fmt.Sprintf("%T", v), tt.wantGo)
			u, ok := v.(*Unknown)
			assert.True(t, ok)
			if tt.wantTop != "" {
				assert.HasKey(t, u.Top, tt.wantTop)
			}
		})
	}
}

func TestParse_SystemInit_Spot(t *testing.T) {
	t.Parallel()
	lines := schematest.Lines(t, "sample.jsonl")
	v, err := Parse([]byte(lines[0]))
	assert.ErrVal(t, err).Nil().Fatal()

	init, ok := v.(*SystemInit)
	assert.True(t, ok)
	assert.NotZero(t, init.Model)
	if !slices.Contains(init.Tools, "Bash") {
		t.Errorf("Tools missing Bash: %v", init.Tools)
	}
	assert.NotZero(t, init.CWD)
}

func TestParse_Assistant_ToolUse_Spot(t *testing.T) {
	t.Parallel()
	lines := schematest.Lines(t, "sample.jsonl")
	// Line 3 (index 2) is the assistant/tool_use fixture.
	v, err := Parse([]byte(lines[2]))
	assert.ErrVal(t, err).Nil().Fatal()

	a, ok := v.(*Assistant)
	assert.True(t, ok)
	assert.Eq(t, len(a.Message.Content), 1)
	block := a.Message.Content[0]
	assert.Eq(t, block.Type, "tool_use")
	assert.NotZero(t, block.Name)
	assert.NotZero(t, len(block.Input))
}

func TestParse_Raw_PreservesUnmodelled(t *testing.T) {
	t.Parallel()
	lines := schematest.Lines(t, "sample.jsonl")
	v, err := Parse([]byte(lines[0]))
	assert.ErrVal(t, err).Nil().Fatal()

	init, ok := v.(*SystemInit)
	assert.True(t, ok)
	var top map[string]json.RawMessage
	assert.ErrVal(t, json.Unmarshal(init.Raw, &top)).Nil().Fatal()
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
