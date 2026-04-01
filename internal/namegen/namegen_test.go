// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package namegen

import (
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestGenerate(t *testing.T) {
	for i := 0; i < 10; i++ {
		name := Generate()
		parts := strings.Split(name, "-")
		assert.Equal(t, len(parts), 3)
		if parts[0] == parts[1] {
			t.Errorf("adjectives should be distinct: %q", name)
		}
		t.Log(name)
	}
}
