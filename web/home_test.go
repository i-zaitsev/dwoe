// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestHome_returnsPage(t *testing.T) {
	serv, _ := newTestServer()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	serv.handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, rr.Header().Get("Content-Type"), "text/html; charset=utf-8")
}
