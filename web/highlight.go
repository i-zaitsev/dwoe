// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"html"
	"strings"
)

func highlightJSON(s string) string {
	var buf strings.Builder
	buf.Grow(len(s) * 2)
	i := 0
	for i < len(s) {
		ch := s[i]
		switch {
		case ch == '{' || ch == '}':
			buf.WriteString(`<span class="jb">`)
			buf.WriteByte(ch)
			buf.WriteString(`</span>`)
			i++
		case ch == '[' || ch == ']':
			buf.WriteString(`<span class="jk">`)
			buf.WriteByte(ch)
			buf.WriteString(`</span>`)
			i++
		case ch == ':' || ch == ',':
			buf.WriteString(`<span class="jp">`)
			buf.WriteByte(ch)
			buf.WriteString(`</span>`)
			i++
		case ch == '"':
			j := i + 1
			for j < len(s) {
				if s[j] == '\\' {
					j += 2
					continue
				}
				if s[j] == '"' {
					j++
					break
				}
				j++
			}
			token := s[i:j]

			class := "js"
			k := j
			for k < len(s) && (s[k] == ' ' || s[k] == '\t') {
				k++
			}
			if k < len(s) && s[k] == ':' {
				class = "jkey"
			}

			buf.WriteString(`<span class="`)
			buf.WriteString(class)
			buf.WriteString(`">`)
			buf.WriteString(html.EscapeString(token))
			buf.WriteString(`</span>`)
			i = j
		case ch == 't' || ch == 'f' || ch == 'n':
			j := i
			for j < len(s) && s[j] >= 'a' && s[j] <= 'z' {
				j++
			}
			buf.WriteString(`<span class="jl">`)
			buf.WriteString(s[i:j])
			buf.WriteString(`</span>`)
			i = j
		case (ch >= '0' && ch <= '9') || ch == '-':
			j := i + 1
			for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.' || s[j] == 'e' || s[j] == 'E' || s[j] == '+' || s[j] == '-') {
				j++
			}
			buf.WriteString(`<span class="jn">`)
			buf.WriteString(s[i:j])
			buf.WriteString(`</span>`)
			i = j
		default:
			buf.WriteByte(ch)
			i++
		}
	}
	return buf.String()
}
