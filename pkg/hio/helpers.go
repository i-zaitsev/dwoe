// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type envelope map[string]any

// writeJSON writes body as a JSON response with the given status and extra headers.
func (app *App) writeJSON(w http.ResponseWriter, status int, body envelope, headers http.Header) error {
	js, err := json.Marshal(body)
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for k, vals := range headers {
		for _, val := range vals {
			w.Header().Add(k, val)
		}
	}

	respond(w, status, "application/json", js)

	return nil
}

func respond(w http.ResponseWriter, status int, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeHTML(w http.ResponseWriter, status int, body []byte) {
	respond(w, status, "text/html; charset=utf-8", body)
}

// readRef returns the workspace or batch reference from the "id" path value.
func readRef(r *http.Request) string {
	return r.PathValue("id")
}

func readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	return s
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Boosted") != "true"
}

// isFragment reports whether the request comes from htmx and targets the element with the given ID.
func isFragment(r *http.Request, target string) bool {
	return isHTMX(r) && r.Header.Get("HX-Target") == target
}
