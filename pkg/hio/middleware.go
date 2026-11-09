// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"fmt"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status  int
	started bool
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.started = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	w.started = true
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) FlushError() error {
	w.started = true
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *statusWriter) Flush() {
	_ = w.FlushError()
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (app *App) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if pv := recover(); pv != nil {
				err := fmt.Errorf("%v", pv)
				if sw.started {
					app.logError(r, err)
					return
				}
				w.Header().Set("Connection", "close")
				app.serverError(w, r, err)
			}
		}()
		next.ServeHTTP(sw, r)
	})
}

func (app *App) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		app.Log().Info("request",
			"method", r.Method,
			"uri", r.URL.RequestURI(),
			"status", sw.status,
			"duration", time.Since(start),
		)
	})
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func cacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
