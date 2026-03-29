// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package namegen

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	for i := 0; i < 10; i++ {
		name := Generate()
		parts := strings.Split(name, "-")
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts, got %d: %q", len(parts), name)
		}
		if parts[0] == parts[1] {
			t.Errorf("adjectives should be distinct: %q", name)
		}
		t.Log(name)
	}
}
