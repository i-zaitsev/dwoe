// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"testing"

	"github.com/i-zaitsev/pkg/assert"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

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
			assert.Eq(t, m(tt.input), tt.want)
		})
	}
}

func TestNewEventMatcher(t *testing.T) {
	ev := data.LogEvent{Kind: "tool_use", Text: "Bash: ls", Detail: "total 24"}
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{"empty", "", true},
		{"kind", "tool_use", true},
		{"text", "bash", true},
		{"detail", "total", true},
		{"miss", "nothing", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Eq(t, newEventMatcher(tt.pattern)(ev), tt.want)
		})
	}
}
