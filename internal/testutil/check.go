// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MkdirAll creates a directory path, failing the test on error.
func MkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// FileExists reports whether filename exists on disk.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

// ContainsAll fails if got does not contain every substring.
func ContainsAll(t *testing.T, got string, substrings ...string) {
	t.Helper()
	for _, want := range substrings {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want substring %q", got, want)
		}
	}
}

// DirCount returns the number of entries matching the glob pattern in dir.
func DirCount(dir string, pattern ...string) int {
	dirs, err := filepath.Glob(filepath.Join(dir, filepath.Join(pattern...)))
	if err != nil {
		return -1
	}
	return len(dirs)
}
