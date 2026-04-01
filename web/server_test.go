// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestServer_New(t *testing.T) {
	serv := NewServer(addr)
	assert.Equal(t, serv.addr, addr)
}
