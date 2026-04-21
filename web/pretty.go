// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"bytes"
	"encoding/json"
	"html"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/i-zaitsev/dwoe/schema"
)

const longLineThreshold = 120

func shouldExpand(first, full string) bool {
	return full != first || len(full) > longLineThreshold
}

// prettyRow is the data passed to the pretty-row and pretty-row-expand templates.
// An empty Pretty selects the plain row. Open preopens the details element.
type prettyRow struct {
	Time      string
	Kind      string
	KindClass string
	Body      template.HTML
	Pretty    template.HTML
	Open      bool
}

// logLine is implemented by every per-kind renderer.
// rows returns the prettyRow values a single JSONL event produces.
type logLine interface {
	rows() []prettyRow
}

func renderPrettyLogLine(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	var elem logLine
	v, err := schema.Parse([]byte(trimmed))
	if err != nil || v == nil {
		elem = &containerLine{text: text}
	} else {
		switch m := v.(type) {
		case *schema.SystemInit:
			elem = &initLine{m: m}
		case *schema.TaskStarted:
			elem = &taskStartedLine{m: m}
		case *schema.TaskProgress:
			elem = &taskProgressLine{m: m}
		case *schema.TaskNotification:
			elem = &taskNotificationLine{m: m}
		case *schema.APIRetry:
			elem = &apiRetryLine{m: m}
		case *schema.RateLimitEvent:
			elem = &rateLimitLine{m: m}
		case *schema.Assistant:
			elem = &assistantLine{m: m}
		case *schema.User:
			elem = &userLine{m: m}
		case *schema.Result:
			elem = &resultLine{m: m}
		case *schema.Unknown:
			elem = &unknownLine{m: m}
		default:
			elem = &containerLine{text: text}
		}
	}
	var buf strings.Builder
	writeRows(&buf, elem.rows())
	return buf.String()
}

func writeRows(w io.Writer, rows []prettyRow) {
	for _, r := range rows {
		if r.Pretty == "" {
			writeRow(w, r)
		} else {
			writeExpand(w, r)
		}
	}
}

func writeRow(w io.Writer, row prettyRow) {
	writeTemplate(w, "pretty-row", row)
}

func writeExpand(w io.Writer, row prettyRow) {
	writeTemplate(w, "pretty-row-expand", row)
}

// containerLine renders non-JSON container output as a muted gray row.
// Used as the fallback when schema.Parse fails or returns an unmapped kind.
type containerLine struct{ text string }

func (l *containerLine) rows() []prettyRow {
	t, body := splitBracketTime(l.text)
	return []prettyRow{{
		Time:      t,
		Kind:      "cont",
		KindClass: "cont",
		Body:      template.HTML(html.EscapeString(body)),
	}}
}

// initLine renders system/init events: session_id, model, cwd,
// tool count, and permission mode.
type initLine struct{ m *schema.SystemInit }

func (l *initLine) rows() []prettyRow {
	body := executeTemplate("body-init", struct {
		SessionID, Model, CWD, Permission string
		Tools                             int
	}{
		SessionID:  l.m.SessionID,
		Model:      l.m.Model,
		CWD:        l.m.CWD,
		Permission: l.m.PermissionMode,
		Tools:      len(l.m.Tools),
	})
	return []prettyRow{{
		Time:      envTime(&l.m.Envelope),
		Kind:      "init",
		KindClass: "sys",
		Body:      body,
		Pretty:    prettyJSON(l.m.Envelope.Raw),
	}}
}

// taskStartedLine renders system/task_started events: subagent_type
// and description.
type taskStartedLine struct{ m *schema.TaskStarted }

func (l *taskStartedLine) rows() []prettyRow {
	body := executeTemplate("body-task-started", struct {
		TaskType, Description string
	}{
		TaskType:    l.m.TaskType,
		Description: l.m.Description,
	})
	return []prettyRow{{
		Time:      envTime(&l.m.Envelope),
		Kind:      "task",
		KindClass: "sys",
		Body:      body,
		Pretty:    prettyJSON(l.m.Envelope.Raw),
	}}
}

// taskProgressLine renders system/task_progress events: last tool,
// tool-use count, and token usage.
type taskProgressLine struct{ m *schema.TaskProgress }

func (l *taskProgressLine) rows() []prettyRow {
	data := struct {
		LastTool, Description string
		ToolUses, TotalTokens int
	}{
		LastTool:    l.m.LastToolName,
		Description: l.m.Description,
	}
	if l.m.Usage != nil {
		data.ToolUses = l.m.Usage.ToolUses
		data.TotalTokens = l.m.Usage.TotalTokens
	}
	return []prettyRow{{
		Time:      envTime(&l.m.Envelope),
		Kind:      "task·prog",
		KindClass: "sys",
		Body:      executeTemplate("body-task-progress", data),
	}}
}

// taskNotificationLine renders system/task_notification events:
// final status and summary.
type taskNotificationLine struct{ m *schema.TaskNotification }

func (l *taskNotificationLine) rows() []prettyRow {
	body := executeTemplate("body-task-notification", struct {
		Status, Summary string
	}{
		Status:  l.m.Status,
		Summary: l.m.Summary,
	})
	return []prettyRow{{
		Time:      envTime(&l.m.Envelope),
		Kind:      "task·done",
		KindClass: "sys",
		Body:      body,
		Pretty:    prettyJSON(l.m.Envelope.Raw),
	}}
}

// apiRetryLine renders system/api_retry events: attempt count,
// backoff delay, and error status.
type apiRetryLine struct{ m *schema.APIRetry }

func (l *apiRetryLine) rows() []prettyRow {
	data := struct {
		Attempt, MaxRetries, ErrorStatus int
		HasStatus                        bool
		Delay                            float64
		Error                            string
	}{
		Attempt:    l.m.Attempt,
		MaxRetries: l.m.MaxRetries,
		Delay:      l.m.RetryDelayMs,
		Error:      l.m.Error,
	}
	if l.m.ErrorStatus != nil {
		data.HasStatus = true
		data.ErrorStatus = *l.m.ErrorStatus
	}
	return []prettyRow{{
		Time:      envTime(&l.m.Envelope),
		Kind:      "retry",
		KindClass: "rate",
		Body:      executeTemplate("body-api-retry", data),
	}}
}

// rateLimitLine renders rate_limit_event messages: current status,
// window type, and reset time.
type rateLimitLine struct{ m *schema.RateLimitEvent }

func (l *rateLimitLine) rows() []prettyRow {
	info := l.m.RateLimitInfo
	data := struct {
		Status, Type, Resets string
	}{
		Status: info.Status,
		Type:   info.RateLimitType,
	}
	if info.ResetsAt > 0 {
		data.Resets = time.Unix(info.ResetsAt, 0).Format("15:04")
	}
	return []prettyRow{{
		Time:      envTime(&l.m.Envelope),
		Kind:      "rate",
		KindClass: "rate",
		Body:      executeTemplate("body-rate-limit", data),
	}}
}

// assistantLine renders type=assistant messages. One event fans into one row
// per content block (text, thinking, tool_use), plus an err row if the message
// carried a top-level error.
type assistantLine struct{ m *schema.Assistant }

func (l *assistantLine) rows() []prettyRow {
	tm := envTime(&l.m.Envelope)
	var rows []prettyRow
	for _, block := range l.m.Message.Content {
		switch block.Type {
		case "text":
			first := firstLine(block.Text)
			full := strings.TrimSpace(block.Text)
			row := prettyRow{
				Time:      tm,
				Kind:      "ast·txt",
				KindClass: "ast",
				Body:      template.HTML(html.EscapeString(first)),
			}
			if shouldExpand(first, full) {
				row.Pretty = template.HTML(html.EscapeString(block.Text))
			}
			rows = append(rows, row)
		case "thinking":
			full := strings.TrimSpace(block.Thinking)
			first := firstLine(block.Thinking)
			row := prettyRow{
				Time:      tm,
				Kind:      "think",
				KindClass: "think",
				Body:      template.HTML(html.EscapeString(first)),
			}
			if shouldExpand(first, full) {
				row.Pretty = template.HTML(html.EscapeString(block.Thinking))
			}
			rows = append(rows, row)
		case "tool_use":
			body := executeTemplate("body-tool-use", struct {
				Name, Summary string
			}{
				Name:    block.Name,
				Summary: toolSummary(block.Name, block.Input),
			})
			rows = append(rows, prettyRow{
				Time:      tm,
				Kind:      "tool",
				KindClass: "tool",
				Body:      body,
				Pretty:    prettyJSON(block.Input),
			})
		default:
			rows = append(rows, prettyRow{
				Time:      tm,
				Kind:      block.Type,
				KindClass: "ast",
				Body:      template.HTML(html.EscapeString(block.Text)),
			})
		}
	}
	if l.m.Error != "" {
		rows = append(rows, prettyRow{
			Time:      tm,
			Kind:      "err",
			KindClass: "err",
			Body:      executeTemplate("body-err", struct{ Text string }{l.m.Error}),
		})
	}
	return rows
}

// userLine renders type=user messages. tool_result blocks fan into stdout rows
// on success or err rows when is_error is true. Other text blocks become sys rows.
type userLine struct{ m *schema.User }

func (l *userLine) rows() []prettyRow {
	tm := envTime(&l.m.Envelope)
	var rows []prettyRow
	for _, block := range l.m.Message.Content {
		if block.Type != "tool_result" {
			if block.Text != "" {
				rows = append(rows, prettyRow{
					Time:      tm,
					Kind:      block.Type,
					KindClass: "sys",
					Body:      template.HTML(html.EscapeString(block.Text)),
				})
			}
			continue
		}
		txt := toolResultText(block.Content)
		full := strings.TrimSpace(txt)
		first := firstLine(txt)
		if block.IsError {
			row := prettyRow{
				Time:      tm,
				Kind:      "err",
				KindClass: "err",
				Body:      executeTemplate("body-err", struct{ Text string }{first}),
			}
			if shouldExpand(first, full) {
				row.Pretty = template.HTML(html.EscapeString(txt))
				row.Open = true
			}
			rows = append(rows, row)
			continue
		}
		row := prettyRow{
			Time:      tm,
			Kind:      "stdout",
			KindClass: "stdout",
			Body:      executeTemplate("body-stdout", struct{ Text string }{first}),
		}
		if shouldExpand(first, full) {
			row.Pretty = template.HTML(html.EscapeString(txt))
		}
		rows = append(rows, row)
	}
	return rows
}

// resultLine renders type=result summaries: outcome, cost, duration,
// turn count, and token totals. The row is pre-opened.
type resultLine struct{ m *schema.Result }

func (l *resultLine) rows() []prettyRow {
	outcome := "success"
	if l.m.IsError {
		outcome = "error"
	}
	data := struct {
		Outcome                                      string
		Cost                                         float64
		Mins, Secs, Turns, InputTokens, OutputTokens int
		CacheRead, CacheWrite                        int
		HasUsage                                     bool
	}{
		Outcome: outcome,
		Cost:    l.m.TotalCostUSD,
		Mins:    l.m.DurationMs / 60000,
		Secs:    (l.m.DurationMs / 1000) % 60,
		Turns:   l.m.NumTurns,
	}
	if l.m.Usage != nil {
		data.HasUsage = true
		data.InputTokens = l.m.Usage.InputTokens
		data.OutputTokens = l.m.Usage.OutputTokens
		data.CacheRead = l.m.Usage.CacheReadInputTokens
		data.CacheWrite = l.m.Usage.CacheCreationInputTokens
	}
	return []prettyRow{{
		Time:      envTime(&l.m.Envelope),
		Kind:      "result",
		KindClass: "res",
		Body:      executeTemplate("body-result", data),
		Pretty:    prettyJSON(l.m.Envelope.Raw),
		Open:      true,
	}}
}

// unknownLine renders schema.Unknown values: messages whose type and subtype
// the schema package does not model. Summary shows type and subtype;
// the expanded pane shows the raw JSON.
type unknownLine struct{ m *schema.Unknown }

func (l *unknownLine) rows() []prettyRow {
	body := executeTemplate("body-unknown", struct {
		Type, Subtype string
	}{
		Type:    l.m.Envelope.Type,
		Subtype: l.m.Envelope.Subtype,
	})
	return []prettyRow{{
		Time:      envTime(&l.m.Envelope),
		Kind:      "unknown",
		KindClass: "sys",
		Body:      body,
		Pretty:    prettyJSON(l.m.Envelope.Raw),
	}}
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

func prettyJSON(raw json.RawMessage) template.HTML {
	if len(raw) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return template.HTML(html.EscapeString(string(raw)))
	}
	return template.HTML(highlightJSON(buf.String()))
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
