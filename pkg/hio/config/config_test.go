// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"log/slog"
	"testing"

	"github.com/i-zaitsev/pkg/assert"
)

func TestParse_Defaults(t *testing.T) {
	cfg, err := Parse(nil)

	assert.ErrVal(t, err).Nil().Fatal()
	env := NewEnv(cfg)
	assert.Eq(t, env.ServerAddr(), "127.0.0.1:8080")
	assert.Eq(t, env.LogLevel(), "INFO")
	assert.Eq(t, env.UIDir(), "")
	assert.NotZero(t, env.DataDir())
}

func TestParse_Overrides(t *testing.T) {
	tests := []struct {
		name string
		args []string
		get  func(*Env) string
		want string
	}{
		{"port_only", []string{"-port", "9000"}, (*Env).ServerAddr, "127.0.0.1:9000"},
		{"addr_only", []string{"-addr", "0.0.0.0"}, (*Env).ServerAddr, "0.0.0.0:8080"},
		{"addr_and_port", []string{"-addr", "::1", "-port", "9000"}, (*Env).ServerAddr, "[::1]:9000"},
		{"empty_addr_listens_everywhere", []string{"-addr", "", "-port", "9000"}, (*Env).ServerAddr, ":9000"},
		{"datadir", []string{"-datadir", "/tmp/dwoe"}, (*Env).DataDir, "/tmp/dwoe"},
		{"ui_dir", []string{"-ui-dir", "pkg/hio/ui"}, (*Env).UIDir, "pkg/hio/ui"},
		{"level_debug", []string{"-log-level", "debug"}, (*Env).LogLevel, "DEBUG"},
		{"level_warn", []string{"-log-level", "warn"}, (*Env).LogLevel, "WARN"},
		{"level_error", []string{"-log-level", "error"}, (*Env).LogLevel, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse(tt.args)

			assert.ErrVal(t, err).Nil().Fatal()
			assert.Eq(t, tt.get(NewEnv(cfg)), tt.want)
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"unknown_flag", []string{"-bogus"}},
		{"bad_level", []string{"-log-level", "loud"}},
		{"missing_value", []string{"-addr"}},
		{"port_not_a_number", []string{"-port", "http"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.args)

			assert.ErrVal(t, err).NotNil().Fatal()
		})
	}
}

func TestNewEnv_FromLiteralConfig(t *testing.T) {
	env := NewEnv(Config{Addr: "127.0.0.1", Port: 9090, UIDir: "pkg/hio/ui", LogLevel: slog.LevelWarn})

	assert.Eq(t, env.ServerAddr(), "127.0.0.1:9090")
	assert.Eq(t, env.UIDir(), "pkg/hio/ui")
	assert.Eq(t, env.LogLevel(), "WARN")
}

func TestEnv_LoggerHonoursLevel(t *testing.T) {
	cfg, err := Parse([]string{"-log-level", "error"})
	assert.ErrVal(t, err).Nil().Fatal()

	log := NewEnv(cfg).AppLog()

	assert.False(t, log.Enabled(t.Context(), slog.LevelInfo))
	assert.True(t, log.Enabled(t.Context(), slog.LevelError))
}
