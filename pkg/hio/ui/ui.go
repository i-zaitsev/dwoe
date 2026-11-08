// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package ui embeds the templates and static files of the dashboard.
//
// htmx.min.js and htmx-ext-sse.min.js are vendored from htmx 2.0.4.
package ui

import "embed"

// FS contains the "templates" and "static" directories.
//
//go:embed templates static
var FS embed.FS
