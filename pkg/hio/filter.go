// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"regexp"
	"strings"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

// newLineMatcher returns a case-insensitive matcher.
// For a regular expression or for a plain substring when the pattern is
// not a valid expression.
func newLineMatcher(pattern string) func(string) bool {
	if pattern == "" {
		return func(string) bool { return true }
	}
	if re, err := regexp.Compile("(?i)" + pattern); err == nil {
		return re.MatchString
	}
	lower := strings.ToLower(pattern)
	return func(s string) bool { return strings.Contains(strings.ToLower(s), lower) }
}

// newEventMatcher is a matcher for log events by kind, text, or detail.
func newEventMatcher(pattern string) func(data.LogEvent) bool {
	match := newLineMatcher(pattern)
	return func(ev data.LogEvent) bool {
		return match(ev.Kind + " " + ev.Text + " " + ev.Detail)
	}
}
