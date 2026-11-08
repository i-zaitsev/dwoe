// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"net/http"
	"time"

	"github.com/i-zaitsev/dwoe/pkg/hio/config"
)

// NewServer creates the HTTP server for handler.
// It sets no write timeout because log streams are long-lived.
func NewServer(env *config.Env, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:        env.ServerAddr(),
		Handler:     handler,
		IdleTimeout: time.Minute,
		ReadTimeout: 5 * time.Second,
		ErrorLog:    env.ErrorLog(),
	}
}
