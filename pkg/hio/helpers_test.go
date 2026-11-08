// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/i-zaitsev/pkg/assert"

	"github.com/i-zaitsev/dwoe/pkg/hio/config"
	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

// testApp is an App without models.
// Tests only reach code paths that don't read data, such as rendering,
// errors, middleware, and static files.
type testApp struct {
	*App
	logs *bytes.Buffer
}

func newTestApp(t *testing.T, args ...string) *testApp {
	t.Helper()

	cfg, err := config.Parse(args)
	assert.ErrVal(t, err).Nil().Fatal()

	logs := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	app, err := NewApp(config.NewEnvWithLogger(cfg, logger), data.Models{})
	assert.ErrVal(t, err).Nil().Fatal()

	return &testApp{App: app, logs: logs}
}

// do serves a GET request through the whole handler chain.
func (app *testApp) do(url string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	app.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, url, nil))
	return rr
}
