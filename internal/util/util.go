// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package util provides small general-purpose helpers used across the project.
package util

import (
	"fmt"
	"os"
)

// WriteFileAtomic writes data to a path using a write-then-rename swap.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	return os.Rename(tmp, path)
}
