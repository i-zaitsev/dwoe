// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package data_test

import (
	"testing"

	"github.com/i-zaitsev/pkg/assert"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

func TestWorkspace_Running(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"running", true},
		{"completed", false},
		{"pending", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Eq(t, data.Workspace{Status: tt.status}.Running(), tt.want)
		})
	}
}
