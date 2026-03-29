// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package workspace

type DiffInfo struct {
	Commits []CommitInfo
	Stat    string
	Diff    string
}

type CommitInfo struct {
	Hash    string
	Message string
}
