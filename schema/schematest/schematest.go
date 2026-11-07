// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package schematest provides the stream-json fixtures for tests of the
// schema package and its users. The fixtures are embedded, so tests read
// them by name instead of by a path relative to their own package.
package schematest

import (
	"bufio"
	"bytes"
	"embed"
	"testing"
)

//go:embed testdata/*.jsonl
var fixtures embed.FS

// Lines returns the lines of the named fixture, e.g. "sample.jsonl".
func Lines(t testing.TB, name string) []string {
	t.Helper()
	data, err := fixtures.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}

	var out []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
