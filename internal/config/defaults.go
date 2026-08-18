// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package config defines task and global configuration types, loading, and defaults.
package config

const (
	DefaultMaxTurns   = 200
	DefaultCPUs       = "4"
	DefaultMemory     = "8G"
	DefaultImage      = "dwoe-agent:latest"
	DefaultProxyImage = "dwoe-proxy:latest"
	DefaultProxyPort  = 3128
)

var DefaultAllowList = []string{
	".npmjs.org",
	".npmjs.com",
	".yarnpkg.com",
	".unpkg.com",
	".jsdelivr.net",
	".esm.sh",
	".skypack.dev",
	".cdnjs.cloudflare.com",
	".githubusercontent.com",
	".github.com",
	"developer.mozilla.org",
	".conan.io",
	".vcpkg.io",
	"apt.llvm.org",
	".cppreference.com",
	"proxy.golang.org",
	"sum.golang.org",
	"pkg.go.dev",
	".pypi.org",
	".pythonhosted.org",
	".crates.io",
}

var DefaultPermissions = []string{
	"Bash(*)",
	"Read(*)",
	"Write(*)",
	"Edit(*)",
	"WebFetch(*)",
	"WebSearch(*)",
}
