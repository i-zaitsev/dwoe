// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/i-zaitsev/dwoe/internal/schema"
)

const longLineThreshold = 120

func shouldExpand(first, full string) bool {
	return full != first || len(full) > longLineThreshold
}

func splitBracketTime(text string) (string, string) {
	if len(text) < 11 || text[0] != '[' || text[9] != ']' {
		return "", text
	}
	candidate := text[1:9]
	for i, c := range candidate {
		if i == 2 || i == 5 {
			if c != ':' {
				return "", text
			}
			continue
		}
		if c < '0' || c > '9' {
			return "", text
		}
	}
	return candidate, strings.TrimLeft(text[10:], " ")
}

func envTime(env *schema.Envelope) string {
	if env.Timestamp == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, env.Timestamp)
	if err != nil {
		t, err = time.Parse(time.RFC3339, env.Timestamp)
		if err != nil {
			return ""
		}
	}
	return t.UTC().Format("15:04:05")
}

func toolSummary(name string, input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(input, &m); err != nil {
		return ""
	}
	pick := func(keys ...string) string {
		for _, k := range keys {
			if s, ok := m[k].(string); ok && s != "" {
				return s
			}
		}
		return ""
	}
	switch name {
	case "Bash":
		return pick("command")
	case "Read", "Write", "Edit", "NotebookEdit":
		return pick("file_path")
	case "Grep", "Glob":
		return pick("pattern")
	case "Task":
		return pick("description")
	case "WebFetch":
		return pick("url")
	}
	return ""
}

func toolResultText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var arr []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &arr); err == nil {
		var parts []string
		for _, b := range arr {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return string(raw)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
