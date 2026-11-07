// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package schematest

import (
	"slices"
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestLines_sample(t *testing.T) {
	lines := Lines(t, "sample.jsonl")

	assert.Equal(t, len(lines), 12)
	for _, line := range lines {
		assert.Condition(t, strings.HasPrefix(line, "{"))
	}
}

func TestLines_faulty_keepsBlankLines(t *testing.T) {
	lines := Lines(t, "faulty.jsonl")

	assert.Equal(t, len(lines), 7)
	assert.Condition(t, slices.Contains(lines, ""))
}
