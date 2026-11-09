// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

var funcMap = template.FuncMap{
	"shortID":     shortID,
	"fmtTime":     fmtTime,
	"duration":    duration,
	"statusClass": statusClass,
	"eventClass":  eventClass,
	"diffLines":   diffLines,
	"prettyJSON":  prettyJSON,
}

// duration describes how long a workspace ran.
func duration(w data.Workspace) string {
	if w.StartedAt == nil {
		return "not started"
	}
	if w.FinishedAt == nil {
		return "running"
	}
	d := w.FinishedAt.Sub(*w.StartedAt)
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm %ds", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}

func shortID(id string) string {
	const length = 8
	if len(id) > length {
		return id[:length]
	}
	return id
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format(time.DateTime)
}

func statusClass(status string) string {
	switch status {
	case "completed":
		return "ok"
	case "failed":
		return "bad"
	case "running":
		return "run"
	default:
		return "idle"
	}
}

func eventClass(kind string) string {
	switch kind {
	case "error":
		return "err"
	case "tool_use":
		return "tool"
	case "assistant":
		return "ast"
	case "thinking":
		return "think"
	case "result":
		return "res"
	case "container", "stdout", "tool_result":
		return "cont"
	default:
		return "sys"
	}
}

type diffLine struct {
	Class string
	Text  string
}

func diffLines(patch string) []diffLine {
	if patch == "" {
		return nil
	}

	raw := strings.Split(strings.TrimSuffix(patch, "\n"), "\n")
	lines := make([]diffLine, 0, len(raw))
	for _, text := range raw {
		lines = append(lines, diffLine{Class: diffClass(text), Text: text})
	}

	return lines
}

func diffClass(line string) string {
	switch {
	case strings.HasPrefix(line, "+"):
		return "add"
	case strings.HasPrefix(line, "-"):
		return "del"
	case strings.HasPrefix(line, "@@"):
		return "hunk"
	default:
		return ""
	}
}

type jsonToken struct {
	Class string
	Text  string
}

// prettyJSON indents raw JSON and splits it into tokens classed for highlighting.
// Input that is not valid JSON comes back as one plain token.
func prettyJSON(raw []byte) []jsonToken {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return []jsonToken{{Text: string(raw)}}
	}
	s := buf.String()
	var tokens []jsonToken
	for i := 0; i < len(s); {
		end, class := nextJSONToken(s, i)
		tokens = append(tokens, jsonToken{Class: class, Text: s[i:end]})
		i = end
	}
	return tokens
}

func nextJSONToken(s string, i int) (int, string) {
	switch c := s[i]; {
	case c == '"':
		end := i + 1
		for end < len(s) && s[end] != '"' {
			if s[end] == '\\' {
				end++
			}
			end++
		}
		end = min(end+1, len(s))
		if end < len(s) && s[end] == ':' {
			return end, "jkey"
		}
		return end, "js"
	case strings.IndexByte("{}[]", c) >= 0:
		return i + 1, "jb"
	case c == ':' || c == ',':
		return i + 1, "jp"
	case c == 't' || c == 'f' || c == 'n':
		return scanWhile(s, i, func(b byte) bool { return b >= 'a' && b <= 'z' }), "jl"
	case c == '-' || (c >= '0' && c <= '9'):
		return scanWhile(s, i, func(b byte) bool { return strings.IndexByte("+-.eE0123456789", b) >= 0 }), "jn"
	default:
		return max(scanWhile(s, i, func(b byte) bool { return strings.IndexByte(" \t\r\n", b) >= 0 }), i+1), ""
	}
}

func scanWhile(s string, i int, ok func(byte) bool) int {
	for i < len(s) && ok(s[i]) {
		i++
	}
	return i
}
