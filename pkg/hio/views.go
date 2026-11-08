// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
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
