// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package namegen generates random human-readable names for workspaces.
package namegen

import (
	"math/rand/v2"
)

// Generate returns a random name in the form "adjective-adjective-noun".
func Generate() string {
	i := rand.IntN(len(adjectives))
	j := rand.IntN(len(adjectives) - 1)
	if j >= i {
		j++
	}
	n := nouns[rand.IntN(len(nouns))]
	return adjectives[i] + "-" + adjectives[j] + "-" + n
}
