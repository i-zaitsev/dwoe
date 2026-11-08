// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/i-zaitsev/pkg/assert"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

func TestWriteJSON(t *testing.T) {
	app := newTestApp(t)
	rr := httptest.NewRecorder()

	err := app.writeJSON(rr, http.StatusCreated, envelope{"a": 1}, http.Header{"X-Test": {"yes"}})

	assert.ErrVal(t, err).Nil().Fatal()
	assert.Eq(t, rr.Code, http.StatusCreated)
	assert.Eq(t, rr.Header().Get("Content-Type"), "application/json")
	assert.Eq(t, rr.Header().Get("X-Test"), "yes")
	assert.Eq(t, rr.Body.String(), "{\"a\":1}\n")
}

func TestWriteJSON_UnsupportedValue(t *testing.T) {
	app := newTestApp(t)

	err := app.writeJSON(httptest.NewRecorder(), http.StatusOK, envelope{"f": func() {}}, nil)

	assert.ErrVal(t, err).NotNil().Fatal()
}

func TestErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		respond    func(app *testApp, w http.ResponseWriter, r *http.Request)
		wantStatus int
		wantError  string
	}{
		{
			"not_found",
			func(app *testApp, w http.ResponseWriter, r *http.Request) { app.notFoundResponse(w, r) },
			http.StatusNotFound,
			"the requested resource is not found",
		},
		{
			"method_not_allowed",
			func(app *testApp, w http.ResponseWriter, r *http.Request) { app.methodNotAllowedResponse(w, r) },
			http.StatusMethodNotAllowed,
			"the GET method is not supported",
		},
		{
			"bad_request",
			func(app *testApp, w http.ResponseWriter, r *http.Request) {
				app.badRequestResponse(w, r, errors.New("bad input"))
			},
			http.StatusBadRequest,
			"bad input",
		},
		{
			"server_error",
			func(app *testApp, w http.ResponseWriter, r *http.Request) {
				app.serverErrorResponse(w, r, errors.New("secret detail"))
			},
			http.StatusInternalServerError,
			serverErrorMessage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)
			rr := httptest.NewRecorder()

			tt.respond(app, rr, httptest.NewRequest(http.MethodGet, "/api/v1/x", nil))

			var body map[string]string
			assert.ErrVal(t, json.Unmarshal(rr.Body.Bytes(), &body)).Nil().Fatal()
			assert.Eq(t, rr.Code, tt.wantStatus)
			assert.Eq(t, body["error"], tt.wantError)
		})
	}
}

func TestServerErrorResponse_LogsCause(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/x", nil)

	app.serverErrorResponse(httptest.NewRecorder(), req, errors.New("secret detail"))

	assert.ContainsAll(t, app.logs.String(), "secret detail", "/api/v1/x")
}

func TestServerError_ChoosesResponseByRequest(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		htmx     bool
		wantType string
		wantIn   []string
		wantNone []string
	}{
		{"api_gets_json", "/api/v1/x", false, "application/json", []string{`"error"`}, []string{"<html"}},
		{"page_gets_full_page", "/workspaces/x", false, "text/html; charset=utf-8", []string{"<html", "500"}, nil},
		{"htmx_gets_fragment", "/workspaces/x", true, "text/html; charset=utf-8", []string{`class="error"`}, []string{"<html"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			rr := httptest.NewRecorder()

			app.serverError(rr, req, errors.New("secret detail"))

			assert.Eq(t, rr.Code, http.StatusInternalServerError)
			assert.Eq(t, rr.Header().Get("Content-Type"), tt.wantType)
			assert.ContainsAll(t, rr.Body.String(), tt.wantIn...)
			assert.ContainsNone(t, rr.Body.String(), append(tt.wantNone, "secret detail")...)
			assert.Contains(t, app.logs.String(), "secret detail")
		})
	}
}

func TestModelErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not_found", data.ErrRecordNotFound, http.StatusNotFound},
		{"unavailable", data.ErrUnavailable, http.StatusNotFound},
		{"wrapped_not_found", errors.Join(errors.New("ctx"), data.ErrRecordNotFound), http.StatusNotFound},
		{"other", errors.New("disk on fire"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)
			rr := httptest.NewRecorder()

			app.modelErrorResponse(rr, httptest.NewRequest(http.MethodGet, "/api/v1/x", nil), tt.err)

			assert.Eq(t, rr.Code, tt.wantStatus)
		})
	}
}

func TestReadRef(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("id", "ws-1")

	assert.Eq(t, readRef(req), "ws-1")
}

func TestReadString(t *testing.T) {
	qs := url.Values{"q": {"alpha"}, "empty": {""}}
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"present", "q", "alpha"},
		{"empty_uses_default", "empty", "fallback"},
		{"missing_uses_default", "none", "fallback"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Eq(t, readString(qs, tt.key, "fallback"), tt.want)
		})
	}
}

func TestIsFragment(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{"plain_request", nil, false},
		{"htmx_other_target", map[string]string{"HX-Request": "true", "HX-Target": "main"}, false},
		{"htmx_results", map[string]string{"HX-Request": "true", "HX-Target": "results"}, true},
		{"target_without_htmx", map[string]string{"HX-Target": "results"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			assert.Eq(t, isFragment(req, "results"), tt.want)
		})
	}
}
