// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import "net/http"

type pageConfig struct {
	Theme       string
	BatchFilter string
}

func pageConfigFromRequest(r *http.Request) pageConfig {
	theme := "dark"
	if c, err := r.Cookie("theme"); err == nil && c.Value == "light" {
		theme = "light"
	}
	return pageConfig{Theme: theme}
}
