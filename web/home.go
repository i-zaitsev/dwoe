// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"net/http"
)

func (s *Server) setTheme(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query().Get("v")
	if v != "light" {
		v = "dark"
	}
	http.SetCookie(w, &http.Cookie{Name: "theme", Value: v, Path: "/", SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	cfg := pageConfigFromRequest(r)
	cfg.BatchFilter = r.URL.Query().Get("batch")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeTemplate(w, "page.html", cfg)
}
