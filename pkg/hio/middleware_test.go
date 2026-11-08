// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/i-zaitsev/pkg/assert"
)

func TestRecoverPanic_Api(t *testing.T) {
	app := newTestApp(t)
	handler := app.recoverPanic(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/x", nil))

	assert.Eq(t, rr.Code, http.StatusInternalServerError)
	assert.Eq(t, rr.Header().Get("Connection"), "close")
	assert.Contains(t, rr.Body.String(), serverErrorMessage)
	assert.Eq(t, rr.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, app.logs.String(), "boom")
}

func TestRecoverPanic_Page(t *testing.T) {
	app := newTestApp(t)
	handler := app.recoverPanic(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/workspaces/x", nil))

	assert.Eq(t, rr.Code, http.StatusInternalServerError)
	assert.Eq(t, rr.Header().Get("Content-Type"), "text/html; charset=utf-8")
	assert.ContainsAll(t, rr.Body.String(), "500", serverErrorMessage)
	assert.ContainsNone(t, rr.Body.String(), "boom")
}

func TestLogRequest(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus string
	}{
		{"explicit_status", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }, "status=418"},
		{"implicit_status", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) }, "status=200"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)

			app.logRequest(tt.handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/x?a=b", nil))

			assert.ContainsAll(t, app.logs.String(), "method=GET", `uri="/api/v1/x?a=b"`, tt.wantStatus)
		})
	}
}

func TestLogRequest_KeepsFlusher(t *testing.T) {
	app := newTestApp(t)
	var flushable bool
	handler := app.logRequest(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, flushable = w.(http.Flusher)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	assert.True(t, flushable)
}

func TestSecureHeaders(t *testing.T) {
	handler := secureHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Eq(t, rr.Header().Get("X-Content-Type-Options"), "nosniff")
	assert.Eq(t, rr.Header().Get("X-Frame-Options"), "deny")
	assert.Eq(t, rr.Header().Get("Referrer-Policy"), "no-referrer")
}

func TestCacheControl(t *testing.T) {
	handler := cacheControl(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Eq(t, rr.Header().Get("Cache-Control"), "no-store")
}
