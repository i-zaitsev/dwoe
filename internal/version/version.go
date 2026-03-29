// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package version exposes the build version derived from VCS info.
package version

import (
	"runtime/debug"
)

var version string

// Get returns the version from VCS build info, or "dev" if unavailable.
func Get() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	var (
		rev   string
		dirty bool
	)
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if value := s.Value; value == "" {
				return "dev"
			} else if len(value) > 7 {
				rev = value[:7]
			} else {
				rev = value
			}
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}

	if dirty {
		return rev + "-dirty"
	}

	return rev
}
