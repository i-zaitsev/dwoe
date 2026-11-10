// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"testing"

	"github.com/i-zaitsev/pkg/assert"
)

func TestWebCmd_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		args     []string
		wantHost string
		wantPort int
	}{
		{"defaults", nil, "127.0.0.1", 8080},
		{"port", []string{"--port", "8888"}, "127.0.0.1", 8888},
		{"host", []string{"--host", "0.0.0.0"}, "0.0.0.0", 8080},
		{"host_and_port", []string{"--host", "0.0.0.0", "--port", "9000"}, "0.0.0.0", 9000},
		{"addr", []string{"--addr", "0.0.0.0:9000"}, "0.0.0.0", 9000},
		{"addr_ipv6", []string{"--addr", "[::1]:9000"}, "::1", 9000},
		{"ui_dir", []string{"--ui-dir", "pkg/hio/ui"}, "127.0.0.1", 8080},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := new(cmdWeb)
			assert.ErrVal(t, c.Parse(tt.args)).Nil().Fatal()
			assert.Eq(t, c.host, tt.wantHost)
			assert.Eq(t, c.port, tt.wantPort)
		})
	}
}

func TestWebCmd_Parse_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{"addr_with_host", []string{"--addr", "0.0.0.0:9000", "--host", "127.0.0.1"}},
		{"addr_with_port", []string{"--addr", "0.0.0.0:9000", "--port", "8888"}},
		{"addr_without_port", []string{"--addr", "localhost"}},
		{"addr_with_bad_port", []string{"--addr", "localhost:http"}},
		{"unknown_flag", []string{"--nope"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrVal(t, new(cmdWeb).Parse(tt.args)).NotNil()
		})
	}
}
