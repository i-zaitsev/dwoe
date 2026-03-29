// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestHighlightJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		contain string
	}{
		{
			name:    "key",
			input:   `  "name": "value"`,
			contain: `<span class="jkey">&#34;name&#34;</span>`,
		},
		{
			name:    "string_value",
			input:   `  "key": "hello"`,
			contain: `<span class="js">&#34;hello&#34;</span>`,
		},
		{
			name:    "number",
			input:   `  "count": 42`,
			contain: `<span class="jn">42</span>`,
		},
		{
			name:    "negative_number",
			input:   `  "val": -3.14`,
			contain: `<span class="jn">-3.14</span>`,
		},
		{
			name:    "boolean_true",
			input:   `  "flag": true`,
			contain: `<span class="jl">true</span>`,
		},
		{
			name:    "boolean_false",
			input:   `  "flag": false`,
			contain: `<span class="jl">false</span>`,
		},
		{
			name:    "null_value",
			input:   `  "val": null`,
			contain: `<span class="jl">null</span>`,
		},
		{
			name:    "open_brace",
			input:   `{`,
			contain: `<span class="jb">{</span>`,
		},
		{
			name:    "close_brace",
			input:   `}`,
			contain: `<span class="jb">}</span>`,
		},
		{
			name:    "open_bracket",
			input:   `[`,
			contain: `<span class="jk">[</span>`,
		},
		{
			name:    "close_bracket",
			input:   `]`,
			contain: `<span class="jk">]</span>`,
		},
		{
			name:    "colon",
			input:   `  "a": 1`,
			contain: `<span class="jp">:</span>`,
		},
		{
			name:    "comma",
			input:   `  "a": 1,`,
			contain: `<span class="jp">,</span>`,
		},
		{
			name:    "html_escape_in_string",
			input:   `  "msg": "<script>alert(1)</script>"`,
			contain: `&lt;script&gt;`,
		},
		{
			name:    "escaped_quote_in_string",
			input:   `  "msg": "say \"hi\""`,
			contain: `<span class="js">&#34;say \&#34;hi\&#34;&#34;</span>`,
		},
		{
			name:    "whitespace_preserved",
			input:   "{\n  \"a\": 1\n}",
			contain: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightJSON(tt.input)
			assert.Contains(t, got, tt.contain)
		})
	}
}
