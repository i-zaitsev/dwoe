// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"testing"
	"time"

	"github.com/i-zaitsev/pkg/assert"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

var (
	started  = time.Date(2025, time.March, 15, 9, 30, 5, 0, time.UTC)
	finished = started.Add(7*time.Minute + 12*time.Second)
)

func TestTemplates(t *testing.T) {
	alpha := sampleWorkspace("ws-1", "alpha", "completed")
	beta := sampleWorkspace("ws-2", "beta", "running")
	gamma := sampleWorkspace("ws-3", "gamma", "failed")

	bare := data.Workspace{ID: "ws-4", Name: "delta", Status: "pending"}

	detailed := alpha
	detailed.Model = "claude-sonnet-5"
	detailed.Description = "Add a health endpoint"

	darkLayout := layoutData{Title: "Workspaces", Active: "workspaces", Theme: "dark"}
	lightLayout := layoutData{Title: "Batches", Active: "batches", Theme: "light"}

	events := []data.LogEvent{
		{Time: "10:00:00", Kind: "container", Text: "container up"},
		{Time: "10:00:05", Kind: "init", Text: "claude-x, 12 tools"},
		{Time: "10:00:09", Kind: "assistant", Text: "Reading the layout.", Detail: "Reading the layout.\nThen the tests."},
		{Time: "10:00:10", Kind: "tool_use", Text: "Bash: ls -la"},
		{Time: "10:00:11", Kind: "tool_result", Text: "total 24"},
		{Time: "10:00:12", Kind: "error", Text: "<script>alert(1)</script>", Detail: "first\nsecond"},
		{Time: "10:07:12", Kind: "result", Text: "success: 8 turns"},
	}

	diff := data.Diff{
		Commits: []data.Commit{{Hash: "abc1234", Message: "add <feature>"}},
		Stat:    " feature.go | 2 ++",
		Patch:   "@@ -1,2 +1,3 @@\n package main\n-// TODO\n+func feature() {}\n+var x = \"<b>\"",
	}

	batches := []data.Batch{
		{ID: "batch-abcdef12", Workers: 3, StartedAt: &started, FinishedAt: &finished},
		{ID: "batch-0123456789", Workers: 1, StartedAt: &started},
	}

	testCases := []struct {
		name     string
		set      string
		tmpl     string
		data     any
		expected check
	}{
		{
			name: "layout_dark_wires_htmx_theme_and_navigation",
			set:  "workspaces.html",
			tmpl: "base",
			data: workspacesPageData{layoutData: darkLayout},
			expected: has(
				`<html lang="en" data-theme="dark">`,
				`<title>Workspaces · dwoe</title>`,
				`<body hx-boost="true" hx-ext="sse">`,
				`/static/htmx.min.js`,
				`/static/htmx-ext-sse.min.js`,
				`/static/app.css`,
				`/static/app.js`,
				`"responseHandling"`,
				`<a href="/" class="active">Workspaces</a>`,
				`<a href="/batches">Batches</a>`,
				`<a href="/theme?v=light" hx-boost="false">Light</a>`,
			),
		},
		{
			name: "layout_light_marks_batches_active_and_offers_dark",
			set:  "batches.html",
			tmpl: "base",
			data: batchesPageData{layoutData: lightLayout},
			expected: all(
				has(
					`<html lang="en" data-theme="light">`,
					`<a href="/batches" class="active">Batches</a>`,
					`<a href="/theme?v=dark" hx-boost="false">Dark</a>`,
				),
				lacks(`<a href="/" class="active">`),
			),
		},
		{
			name: "workspaces_page_lists_rows_with_status_batch_and_duration",
			set:  "workspaces.html",
			tmpl: "content",
			data: workspacesPageData{layoutData: darkLayout, Workspaces: []data.Workspace{alpha, beta, gamma}},
			expected: has(
				`<div id="results">`,
				`<a href="/workspaces/ws-1">alpha</a>`,
				`<span class="badge ok">completed</span>`,
				`<span class="badge run">running</span>`,
				`<span class="badge bad">failed</span>`,
				`<a href="/?batch=batch-abcdef12">batch-ab</a>`,
				`2025-03-15 09:30:05`,
				`7m 12s`,
			),
		},
		{
			name: "workspaces_page_with_batch_shows_title_and_clear_link",
			set:  "workspaces.html",
			tmpl: "content",
			data: workspacesPageData{layoutData: darkLayout, Batch: "batch-abcdef12"},
			expected: has(
				`in batch batch-ab`,
				`<a href="/">clear</a>`,
				`<input type="hidden" name="batch" value="batch-abcdef12">`,
			),
		},
		{
			name: "workspaces_page_without_workspaces_says_so",
			set:  "workspaces.html",
			tmpl: "content",
			data: workspacesPageData{layoutData: darkLayout},
			expected: all(
				has(`No workspaces found.`),
				lacks(`<table>`, `clear</a>`, `type="hidden"`),
			),
		},
		{
			name: "workspace_table_is_a_fragment_with_one_row_per_workspace",
			set:  partialsSet,
			tmpl: "workspace-table",
			data: workspacesPageData{Workspaces: []data.Workspace{alpha, beta}},
			expected: all(
				fragment,
				has(`<a href="/workspaces/ws-1">alpha</a>`, `<a href="/workspaces/ws-2">beta</a>`),
			),
		},
		{
			name: "workspace_table_leaves_batch_empty_without_one",
			set:  partialsSet,
			tmpl: "workspace-table",
			data: workspacesPageData{Workspaces: []data.Workspace{bare}},
			expected: all(
				has(`<a href="/workspaces/ws-4">delta</a>`, `<span class="badge idle">pending</span>`, `not started`),
				lacks(`?batch=`),
			),
		},
		{
			name: "filter_searches_the_results_with_a_debounce_and_keeps_the_query",
			set:  partialsSet,
			tmpl: "filter",
			data: workspacesPageData{Query: "alp"},
			expected: all(
				fragment,
				has(
					`value="alp"`,
					`hx-get="/"`,
					`hx-include="closest form"`,
					`hx-target="#results"`,
					`hx-push-url="true"`,
					`hx-trigger="input changed delay:300ms, search"`,
					`<noscript><button type="submit">Filter</button></noscript>`,
				),
				lacks(`type="hidden"`),
			),
		},
		{
			name: "workspace_page_running_follows_the_log_and_polls_the_info",
			set:  "workspace.html",
			tmpl: "content",
			data: workspacePageData{layoutData: darkLayout, Workspace: beta},
			expected: all(
				has(
					`hx-get="/workspaces/ws-2/info" hx-trigger="every 5s" hx-swap="outerHTML"`,
					`<div sse-connect="/workspaces/ws-2/logs/stream" sse-close="done">`,
					`<div sse-swap="log,error" hx-swap="beforeend" hx-target="#log-rows"></div>`,
					`<div id="log-rows" class="log-rows" data-follow></div>`,
					`hx-get="/workspaces/ws-2/diff" hx-trigger="load"`,
				),
				lacks(`hx-get="/workspaces/ws-2/logs"`, `name="filter"`),
			),
		},
		{
			name: "workspace_page_finished_loads_the_log_once_and_stops_polling",
			set:  "workspace.html",
			tmpl: "content",
			data: workspacePageData{layoutData: darkLayout, Workspace: alpha},
			expected: all(
				has(
					`<span class="badge ok">completed</span>`,
					`<th>Finished</th><td>2025-03-15 09:37:17</td>`,
					`<th>Duration</th><td>7m 12s</td>`,
					`hx-get="/workspaces/ws-1/logs" hx-trigger="load" hx-swap="innerHTML"`,
					`<input type="search" name="filter"`,
					`hx-get="/workspaces/ws-1/diff" hx-trigger="load"`,
				),
				lacks(`sse-connect`, `every 5s`, `data-follow`),
			),
		},
		{
			name: "workspace_page_escapes_the_prompt_and_shows_the_task_config",
			set:  "workspace.html",
			tmpl: "content",
			data: workspacePageData{layoutData: darkLayout, Workspace: alpha},
			expected: all(
				has(
					`<pre class="code"><code>Do the &lt;b&gt;thing&lt;/b&gt;.</code></pre>`,
					`<h2>Task config</h2>`,
					`name: alpha`,
					`/tmp/ws-base`,
				),
				lacks(`<b>thing</b>`),
			),
		},
		{
			name: "workspace_page_without_prompt_and_config_omits_those_panels",
			set:  "workspace.html",
			tmpl: "content",
			data: workspacePageData{layoutData: darkLayout, Workspace: bare},
			expected: all(
				has(`<h2>delta `, `<h2>Logs</h2>`, `<h2>Changes</h2>`),
				lacks(`<h2>Prompt</h2>`, `<h2>Task config</h2>`, `<th>Batch</th>`),
			),
		},
		{
			name: "workspace_info_polls_only_while_running",
			set:  partialsSet,
			tmpl: "workspace-info",
			data: beta,
			expected: all(
				fragment,
				has(`<section class="panel" hx-get="/workspaces/ws-2/info" hx-trigger="every 5s"`, `<th>Duration</th><td>running</td>`),
			),
		},
		{
			name: "workspace_info_finished_is_static_and_links_the_batch",
			set:  partialsSet,
			tmpl: "workspace-info",
			data: alpha,
			expected: all(
				fragment,
				has(`<section class="panel">`, `<a href="/batches/batch-abcdef12">batch-abcdef12</a>`),
				lacks(`hx-get`, `hx-trigger`),
			),
		},
		{
			name:     "workspace_info_shows_model_and_description_when_set",
			set:      partialsSet,
			tmpl:     "workspace-info",
			data:     detailed,
			expected: has(`<th>Model</th><td>claude-sonnet-5</td>`, `Add a health endpoint`),
		},
		{
			name:     "workspace_info_omits_model_when_unset",
			set:      partialsSet,
			tmpl:     "workspace-info",
			data:     alpha,
			expected: lacks(`<th>Model</th>`),
		},
		{
			name: "diff_lazy_loads_the_diff_when_it_appears",
			set:  partialsSet,
			tmpl: "diff-lazy",
			data: alpha,
			expected: all(
				fragment,
				has(`<div hx-get="/workspaces/ws-1/diff" hx-trigger="load">`, `Loading…`),
			),
		},
		{
			name: "log_rows_use_one_class_per_kind",
			set:  partialsSet,
			tmpl: "log-rows",
			data: events,
			expected: all(
				fragment,
				has(
					`<div class="log cont">`,
					`<div class="log sys">`,
					`<div class="log ast">`,
					`<div class="log tool">`,
					`<div class="log err">`,
					`<div class="log res">`,
					`<span class="time">10:00:00</span>`,
					`<span class="kind">tool_use</span>`,
					`<span class="text">container up</span>`,
				),
			),
		},
		{
			name: "log_rows_expand_events_with_detail_and_escape_their_text",
			set:  partialsSet,
			tmpl: "log-rows",
			data: events,
			expected: all(
				has(
					`<details><summary>Reading the layout.</summary><pre>Reading the layout.`,
					`<details><summary>&lt;script&gt;alert(1)&lt;/script&gt;</summary><pre>first`,
				),
				lacks(`<script>`),
			),
		},
		{
			name:     "log_rows_say_so_when_empty",
			set:      partialsSet,
			tmpl:     "log-rows",
			data:     []data.LogEvent(nil),
			expected: all(has(`No log lines.`), lacks(`class="log`)),
		},
		{
			name: "log_row_without_detail_is_plain_text",
			set:  partialsSet,
			tmpl: "log-row",
			data: events[3],
			expected: all(
				has(`<span class="text">Bash: ls -la</span>`),
				lacks(`<details>`),
			),
		},
		{
			name: "diff_lists_commits_stat_and_classifies_patch_lines",
			set:  partialsSet,
			tmpl: "diff",
			data: diff,
			expected: all(
				fragment,
				has(
					`<code>abc1234</code> add &lt;feature&gt;`,
					`feature.go | 2`,
					`<pre class="diff">`,
					`<span class="hunk">@@ -1,2`,
					`<span class="del">-// TODO`,
					`<span class="add">`,
					`<span class=""> package main`,
				),
				lacks(`<b>`, `<feature>`),
			),
		},
		{
			name: "diff_without_commits_or_stat_shows_only_the_patch",
			set:  partialsSet,
			tmpl: "diff",
			data: data.Diff{Patch: "-old"},
			expected: all(
				has(`<pre class="diff">`, `<span class="del">-old`),
				lacks(`class="commits"`, `class="muted"`),
			),
		},
		{
			name: "error_message_is_a_paragraph",
			set:  partialsSet,
			tmpl: "error",
			data: "Not <available>.",
			expected: all(
				fragment,
				has(`<p class="error">Not &lt;available&gt;.</p>`),
			),
		},
		{
			name: "batches_page_lists_batches_with_workers_and_links",
			set:  "batches.html",
			tmpl: "content",
			data: batchesPageData{layoutData: lightLayout, Batches: batches},
			expected: has(
				`<a href="/?batch=batch-abcdef12">batch-ab</a>`,
				`<td>3</td>`,
				`<td>2025-03-15 09:37:17</td>`,
				`<a href="/batches/batch-abcdef12">Changes</a>`,
				`<a href="/?batch=batch-0123456789">batch-01</a>`,
				`<td>-</td>`,
			),
		},
		{
			name:     "batches_page_without_batches_says_so",
			set:      "batches.html",
			tmpl:     "content",
			data:     batchesPageData{layoutData: lightLayout},
			expected: all(has(`No batches found.`), lacks(`<table>`)),
		},
		{
			name: "batch_page_shows_the_diff_of_every_workspace",
			set:  "batch.html",
			tmpl: "content",
			data: batchPageData{
				layoutData: darkLayout,
				Batch: data.Batch{ID: "batch-abcdef12", Changes: []data.BatchChange{
					{Workspace: alpha, Diff: diff},
					{Workspace: gamma, Diff: data.Diff{Patch: "-gone"}},
				}},
			},
			expected: has(
				`<h2>Changes in batch batch-ab</h2>`,
				`<h3><a href="/workspaces/ws-1">alpha</a></h3>`,
				`<h3><a href="/workspaces/ws-3">gamma</a></h3>`,
				`<span class="del">-// TODO`,
				`<span class="del">-gone`,
			),
		},
		{
			name:     "batch_page_without_changes_says_so",
			set:      "batch.html",
			tmpl:     "content",
			data:     batchPageData{layoutData: darkLayout, Batch: data.Batch{ID: "batch-abcdef12"}},
			expected: all(has(`No changes.`), lacks(`<h3>`)),
		},
		{
			name: "error_page_shows_status_message_and_a_way_back",
			set:  "error.html",
			tmpl: "content",
			data: errorPage{layoutData: darkLayout, Status: 404, StatusText: "Not Found", Message: "the requested page is not found"},
			expected: has(
				`<h2>404 Not Found</h2>`,
				`the requested page is not found`,
				`<a href="/">Back to workspaces</a>`,
			),
		},
		{
			name: "workspace_info_error_keeps_polling",
			set:  partialsSet,
			tmpl: "workspace-info-error",
			data: workspaceInfoError{ID: "ws-2", Message: "try later"},
			expected: has(
				`hx-get="/workspaces/ws-2/info"`,
				`hx-trigger="every 5s"`,
				`hx-swap="outerHTML"`,
				`<p class="error">try later</p>`,
			),
		},
		{
			name:     "log_error_row_shows_escaped_message_with_error_class",
			set:      partialsSet,
			tmpl:     "log-error-row",
			data:     "<b>the log stream failed</b>",
			expected: has(`<div class="log err">`, `<span class="kind">error</span>`, `&lt;b&gt;the log stream failed`),
		},
	}

	app := newTestApp(t)
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := app.templates.execute(tt.set, tt.tmpl, tt.data)
			assert.ErrVal(t, err).Nil().Fatal()
			tt.expected(t, rendered.String())
		})
	}
}

func sampleWorkspace(id, name, status string) data.Workspace {
	ws := data.Workspace{
		ID:         id,
		Name:       name,
		Status:     status,
		Exit:       "pending",
		BasePath:   "/tmp/ws-base",
		BatchID:    "batch-abcdef12",
		StartedAt:  &started,
		Prompt:     "Do the <b>thing</b>.",
		TaskConfig: "name: alpha\n",
	}
	if status == "completed" || status == "failed" {
		ws.FinishedAt = &finished
	}
	return ws
}

// check asserts something about rendered output.
type check func(t *testing.T, s string)

func has(subs ...string) check {
	return func(t *testing.T, s string) {
		t.Helper()
		assert.ContainsAll(t, s, subs...)
	}
}

func lacks(subs ...string) check {
	return func(t *testing.T, s string) {
		t.Helper()
		assert.ContainsNone(t, s, subs...)
	}
}

func all(checks ...check) check {
	return func(t *testing.T, s string) {
		t.Helper()
		for _, c := range checks {
			c(t, s)
		}
	}
}

// fragment checks that the output is a piece of a page, not a whole document.
var fragment = lacks("<!doctype", "<html", "<head", "<body")
