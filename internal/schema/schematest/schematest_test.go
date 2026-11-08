// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package schematest

import (
	"slices"
	"strings"
	"testing"

	"github.com/i-zaitsev/pkg/assert"
)

func TestLines_sample(t *testing.T) {
	lines := Lines(t, "sample.jsonl")

	assert.Eq(t, len(lines), 12)
	for _, line := range lines {
		assert.True(t, strings.HasPrefix(line, "{"))
	}
}

func TestLines_faulty_keepsBlankLines(t *testing.T) {
	lines := Lines(t, "faulty.jsonl")

	assert.Eq(t, len(lines), 7)
	assert.True(t, slices.Contains(lines, ""))
}
