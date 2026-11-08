// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package schema parses Claude Code `claude -p --output-format stream-json`
// output lines into typed Go values. It models only the message shapes
// observed in real workspace logs; everything else falls through to Unknown.
//
// Each concrete type embeds Envelope which carries the original JSON bytes
// in Raw. Renderers use Raw to reach fields that are not modelled on the
// concrete type without requiring changes to this package.
package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrEmpty is returned by Parse when the input line has no JSON content.
var ErrEmpty = errors.New("empty line")

// Envelope holds discriminator fields common to every Claude message.
// It is embedded in every concrete type and in Unknown.
type Envelope struct {
	Type            string          `json:"type"`
	Subtype         string          `json:"subtype,omitempty"`
	SessionID       string          `json:"session_id,omitempty"`
	UUID            string          `json:"uuid,omitempty"`
	ParentToolUseID string          `json:"parent_tool_use_id,omitempty"`
	Timestamp       string          `json:"timestamp,omitempty"`
	Raw             json.RawMessage `json:"-"`
}

// IsSubAgent reports whether the message was produced inside a sub-agent
// conversation spawned by the Task tool.
func (e *Envelope) IsSubAgent() bool {
	return e.ParentToolUseID != ""
}

// SystemInit corresponds to (type=system, subtype=init).
type SystemInit struct {
	Envelope
	CWD               string      `json:"cwd,omitempty"`
	Model             string      `json:"model,omitempty"`
	Tools             []string    `json:"tools,omitempty"`
	MCPServers        []MCPServer `json:"mcp_servers,omitempty"`
	PermissionMode    string      `json:"permissionMode,omitempty"`
	SlashCommands     []string    `json:"slash_commands,omitempty"`
	Agents            []string    `json:"agents,omitempty"`
	Skills            []string    `json:"skills,omitempty"`
	ClaudeCodeVersion string      `json:"claude_code_version,omitempty"`
	APIKeySource      string      `json:"apiKeySource,omitempty"`
}

// MCPServer is one entry in SystemInit.MCPServers.
type MCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

// TaskStarted corresponds to (type=system, subtype=task_started).
type TaskStarted struct {
	Envelope
	TaskID      string `json:"task_id,omitempty"`
	ToolUseID   string `json:"tool_use_id,omitempty"`
	Description string `json:"description,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
}

// TaskProgress corresponds to (type=system, subtype=task_progress).
type TaskProgress struct {
	Envelope
	TaskID       string     `json:"task_id,omitempty"`
	ToolUseID    string     `json:"tool_use_id,omitempty"`
	Description  string     `json:"description,omitempty"`
	LastToolName string     `json:"last_tool_name,omitempty"`
	Usage        *TaskUsage `json:"usage,omitempty"`
}

// TaskNotification corresponds to (type=system, subtype=task_notification).
type TaskNotification struct {
	Envelope
	TaskID     string     `json:"task_id,omitempty"`
	ToolUseID  string     `json:"tool_use_id,omitempty"`
	Status     string     `json:"status,omitempty"`
	Summary    string     `json:"summary,omitempty"`
	OutputFile string     `json:"output_file,omitempty"`
	Usage      *TaskUsage `json:"usage,omitempty"`
}

// TaskUsage is the lightweight usage block on TaskProgress and TaskNotification.
type TaskUsage struct {
	TotalTokens int `json:"total_tokens,omitempty"`
	ToolUses    int `json:"tool_uses,omitempty"`
	DurationMs  int `json:"duration_ms,omitempty"`
}

// APIRetry corresponds to (type=system, subtype=api_retry).
type APIRetry struct {
	Envelope
	Attempt      int     `json:"attempt,omitempty"`
	MaxRetries   int     `json:"max_retries,omitempty"`
	RetryDelayMs float64 `json:"retry_delay_ms,omitempty"`
	ErrorStatus  *int    `json:"error_status,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// Assistant corresponds to type=assistant.
type Assistant struct {
	Envelope
	Message ChatMessage `json:"message"`
	Error   string      `json:"error,omitempty"`
}

// User corresponds to type=user.
type User struct {
	Envelope
	Message       ChatMessage     `json:"message"`
	ToolUseResult json.RawMessage `json:"tool_use_result,omitempty"`
}

// ChatMessage is the inner message payload shared by Assistant and User.
type ChatMessage struct {
	Role       string         `json:"role,omitempty"`
	Model      string         `json:"model,omitempty"`
	Content    []ContentBlock `json:"content,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Usage      *TokenUsage    `json:"usage,omitempty"`
}

// ContentBlock is a flat representation of an Anthropic content block.
// Consumers dispatch on Type; fields that do not apply are empty.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// TokenUsage mirrors the Anthropic API usage block.
type TokenUsage struct {
	InputTokens              int `json:"input_tokens,omitempty"`
	OutputTokens             int `json:"output_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// Result corresponds to type=result. Only result/success is observed today;
// error subtypes would land in Unknown.
type Result struct {
	Envelope
	IsError       bool        `json:"is_error,omitempty"`
	DurationMs    int         `json:"duration_ms,omitempty"`
	DurationAPIMs int         `json:"duration_api_ms,omitempty"`
	NumTurns      int         `json:"num_turns,omitempty"`
	Result        string      `json:"result,omitempty"`
	TotalCostUSD  float64     `json:"total_cost_usd,omitempty"`
	Usage         *TokenUsage `json:"usage,omitempty"`
}

// RateLimitEvent corresponds to type=rate_limit_event.
type RateLimitEvent struct {
	Envelope
	RateLimitInfo RateLimitInfo `json:"rate_limit_info"`
}

// RateLimitInfo is the payload of a RateLimitEvent. Fields vary across
// CLI versions; unknown fields remain reachable via Envelope.Raw.
type RateLimitInfo struct {
	Status                string `json:"status,omitempty"`
	ResetsAt              int64  `json:"resetsAt,omitempty"`
	RateLimitType         string `json:"rateLimitType,omitempty"`
	OverageStatus         string `json:"overageStatus,omitempty"`
	OverageDisabledReason string `json:"overageDisabledReason,omitempty"`
	IsUsingOverage        bool   `json:"isUsingOverage,omitempty"`
}

// Unknown is the catch-all for messages this package does not model.
// Top exposes every top-level key of the original line so a renderer can
// show unmodelled messages generically.
type Unknown struct {
	Envelope
	Top map[string]json.RawMessage
}

// Parse decodes a single JSONL line into a concrete *T or *Unknown.
//
// Return rules:
//   - empty/whitespace-only line returns (nil, ErrEmpty)
//   - non-JSON input returns (nil, err)
//   - valid JSON that does not match any known (type, subtype) returns *Unknown
//   - valid JSON that matches a known type but fails the concrete unmarshal
//     (e.g. message.content is a string instead of an array) falls back to
//     *Unknown rather than erroring, so one edge-case line never breaks a
//     streaming consumer
//
// Every concrete pointer has Envelope.Raw set to line.
func Parse(line []byte) (any, error) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return nil, ErrEmpty
	}

	var env Envelope
	if err := json.Unmarshal(trimmed, &env); err != nil {
		return nil, fmt.Errorf("schema: parse envelope: %w", err)
	}
	env.Raw = append(json.RawMessage(nil), trimmed...)

	key := env.Type
	if env.Subtype != "" {
		key = env.Type + "/" + env.Subtype
	}

	switch key {
	case "system/init":
		return decode[SystemInit](trimmed, env)
	case "system/task_started":
		return decode[TaskStarted](trimmed, env)
	case "system/task_progress":
		return decode[TaskProgress](trimmed, env)
	case "system/task_notification":
		return decode[TaskNotification](trimmed, env)
	case "system/api_retry":
		return decode[APIRetry](trimmed, env)
	case "assistant":
		return decode[Assistant](trimmed, env)
	case "user":
		return decode[User](trimmed, env)
	case "result/success":
		return decode[Result](trimmed, env)
	case "rate_limit_event":
		return decode[RateLimitEvent](trimmed, env)
	}

	return unknown(trimmed, env), nil
}

// decode unmarshals line into *T and sets T.Envelope.Raw. On failure it
// falls back to *Unknown so the caller never sees a decode error for a
// valid JSON object.
func decode[T any](line []byte, env Envelope) (any, error) {
	var v T
	if err := json.Unmarshal(line, &v); err != nil {
		return unknown(line, env), nil
	}
	setRaw(&v, env.Raw)
	return &v, nil
}

// setRaw writes env.Raw into the embedded Envelope of v via reflection-free
// type assertion on the one interface every concrete type satisfies.
func setRaw(v any, raw json.RawMessage) {
	if h, ok := v.(interface{ envelope() *Envelope }); ok {
		h.envelope().Raw = raw
	}
}

func (e *Envelope) envelope() *Envelope { return e }

func unknown(line []byte, env Envelope) *Unknown {
	var top map[string]json.RawMessage
	_ = json.Unmarshal(line, &top)
	return &Unknown{Envelope: env, Top: top}
}
