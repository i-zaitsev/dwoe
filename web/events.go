// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"fmt"
	"strings"
	"time"

	"github.com/i-zaitsev/dwoe/internal/schema"
	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

// parseLogEvents converts a single log line into the events it carries.
// One JSONL message may fan out into several events, for example, one per
// content block of an assistant message.
func parseLogEvents(line string) []data.LogEvent {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	v, err := schema.Parse([]byte(trimmed))
	if err != nil || v == nil {
		return []data.LogEvent{containerEvent(line)}
	}

	switch m := v.(type) {
	case *schema.SystemInit:
		return []data.LogEvent{initEvent(m)}
	case *schema.TaskStarted:
		return []data.LogEvent{taskStartedEvent(m)}
	case *schema.TaskProgress:
		return []data.LogEvent{taskProgressEvent(m)}
	case *schema.TaskNotification:
		return []data.LogEvent{taskNotificationEvent(m)}
	case *schema.APIRetry:
		return []data.LogEvent{apiRetryEvent(m)}
	case *schema.RateLimitEvent:
		return []data.LogEvent{rateLimitEvent(m)}
	case *schema.Assistant:
		return assistantEvents(m)
	case *schema.User:
		return userEvents(m)
	case *schema.Result:
		return []data.LogEvent{resultEvent(m)}
	case *schema.Unknown:
		return []data.LogEvent{unknownEvent(m)}
	}

	return []data.LogEvent{containerEvent(line)}
}

func withDetail(ev data.LogEvent, full string) data.LogEvent {
	if shouldExpand(firstLine(full), strings.TrimSpace(full)) {
		ev.Detail = full
	}
	return ev
}

func containerEvent(text string) data.LogEvent {
	t, body := splitBracketTime(text)
	return data.LogEvent{Time: t, Kind: "container", Text: body}
}

func initEvent(m *schema.SystemInit) data.LogEvent {
	return data.LogEvent{
		Time: envTime(&m.Envelope),
		Kind: "init",
		Text: fmt.Sprintf("%s, %d tools, permission %s, cwd %s", m.Model, len(m.Tools), m.PermissionMode, m.CWD),
		Raw:  m.Envelope.Raw,
	}
}

func taskStartedEvent(m *schema.TaskStarted) data.LogEvent {
	return data.LogEvent{
		Time: envTime(&m.Envelope),
		Kind: "task_started",
		Text: m.TaskType + ": " + m.Description,
		Raw:  m.Envelope.Raw,
	}
}

func taskProgressEvent(m *schema.TaskProgress) data.LogEvent {
	var toolUses, tokens int
	if m.Usage != nil {
		toolUses, tokens = m.Usage.ToolUses, m.Usage.TotalTokens
	}
	return data.LogEvent{
		Time: envTime(&m.Envelope),
		Kind: "task_progress",
		Text: fmt.Sprintf("%s: %s (%d tool uses, %d tokens)", m.LastToolName, m.Description, toolUses, tokens),
	}
}

func taskNotificationEvent(m *schema.TaskNotification) data.LogEvent {
	return data.LogEvent{
		Time: envTime(&m.Envelope),
		Kind: "task_done",
		Text: m.Status + ": " + m.Summary,
		Raw:  m.Envelope.Raw,
	}
}

func apiRetryEvent(m *schema.APIRetry) data.LogEvent {
	text := fmt.Sprintf("attempt %d/%d, retry in %.0fms", m.Attempt, m.MaxRetries, m.RetryDelayMs)
	if m.ErrorStatus != nil {
		text += fmt.Sprintf(" (status %d)", *m.ErrorStatus)
	}
	if m.Error != "" {
		text += ": " + m.Error
	}
	return data.LogEvent{Time: envTime(&m.Envelope), Kind: "api_retry", Text: text}
}

func rateLimitEvent(m *schema.RateLimitEvent) data.LogEvent {
	info := m.RateLimitInfo
	text := fmt.Sprintf("%s (%s)", info.Status, info.RateLimitType)
	if info.ResetsAt > 0 {
		text += ", resets " + time.Unix(info.ResetsAt, 0).Format("15:04")
	}
	return data.LogEvent{Time: envTime(&m.Envelope), Kind: "rate_limit", Text: text}
}

func assistantEvents(m *schema.Assistant) []data.LogEvent {
	tm := envTime(&m.Envelope)

	var events []data.LogEvent
	for _, block := range m.Message.Content {
		switch block.Type {
		case "text":
			ev := data.LogEvent{Time: tm, Kind: "assistant", Text: firstLine(block.Text)}
			events = append(events, withDetail(ev, block.Text))
		case "thinking":
			ev := data.LogEvent{Time: tm, Kind: "thinking", Text: firstLine(block.Thinking)}
			events = append(events, withDetail(ev, block.Thinking))
		case "tool_use":
			text := block.Name
			if summary := toolSummary(block.Name, block.Input); summary != "" {
				text += ": " + summary
			}
			events = append(events, data.LogEvent{Time: tm, Kind: "tool_use", Text: text, Raw: block.Input})
		default:
			events = append(events, data.LogEvent{Time: tm, Kind: block.Type, Text: block.Text})
		}
	}

	if m.Error != "" {
		events = append(events, data.LogEvent{Time: tm, Kind: "error", Text: m.Error})
	}

	return events
}

func userEvents(m *schema.User) []data.LogEvent {
	tm := envTime(&m.Envelope)

	var events []data.LogEvent
	for _, block := range m.Message.Content {
		if block.Type != "tool_result" {
			if block.Text != "" {
				events = append(events, data.LogEvent{Time: tm, Kind: block.Type, Text: block.Text})
			}
			continue
		}

		text := toolResultText(block.Content)
		kind := "tool_result"
		if block.IsError {
			kind = "error"
		}
		events = append(events, withDetail(data.LogEvent{Time: tm, Kind: kind, Text: firstLine(text)}, text))
	}

	return events
}

func resultEvent(m *schema.Result) data.LogEvent {
	outcome := "success"
	if m.IsError {
		outcome = "error"
	}
	mins, secs := m.DurationMs/60000, (m.DurationMs/1000)%60
	ev := data.LogEvent{
		Time: envTime(&m.Envelope),
		Kind: "result",
		Text: fmt.Sprintf("%s: %d turns, %dm%02ds, $%.4f", outcome, m.NumTurns, mins, secs, m.TotalCostUSD),
		Raw:  m.Envelope.Raw,
	}
	return withDetail(ev, m.Result)
}

func unknownEvent(m *schema.Unknown) data.LogEvent {
	text := m.Envelope.Type
	if m.Envelope.Subtype != "" {
		text += "/" + m.Envelope.Subtype
	}
	return data.LogEvent{Time: envTime(&m.Envelope), Kind: "unknown", Text: text, Raw: m.Envelope.Raw}
}
