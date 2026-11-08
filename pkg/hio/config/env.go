// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"log"
	"log/slog"
	"net"
	"os"
	"strconv"
)

// Env is the runtime environment of the server: its settings and logger.
type Env struct {
	log *slog.Logger
	cfg Config
}

// NewEnv creates an Env that logs to stderr at the configured level.
func NewEnv(cfg Config) *Env {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel}))
	return NewEnvWithLogger(cfg, logger)
}

// NewEnvWithLogger creates an Env that logs to the given logger.
func NewEnvWithLogger(cfg Config, logger *slog.Logger) *Env {
	return &Env{log: logger, cfg: cfg}
}

func (e *Env) AppLog() *slog.Logger {
	return e.log
}

func (e *Env) ErrorLog() *log.Logger {
	return slog.NewLogLogger(e.log.Handler(), slog.LevelError)
}

// ServerAddr returns the address to listen on, as host:port.
func (e *Env) ServerAddr() string {
	return net.JoinHostPort(e.cfg.Addr, strconv.Itoa(e.cfg.Port))
}

func (e *Env) DataDir() string {
	return e.cfg.DataDir
}

func (e *Env) UIDir() string {
	return e.cfg.UIDir
}

func (e *Env) LogLevel() string {
	return e.cfg.LogLevel.String()
}
