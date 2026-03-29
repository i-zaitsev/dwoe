// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package state

import "strings"

// NotFoundError is returned when a workspace ID does not exist in the store.
type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string {
	return e.ID + ": not found"
}

// AmbiguousMatchError is returned when an ID prefix matches multiple workspaces.
type AmbiguousMatchError struct {
	Prefix string
	IDs    []string
}

func (e *AmbiguousMatchError) Error() string {
	return e.Prefix + ": ambiguous, matches " + strings.Join(e.IDs, ", ")
}
