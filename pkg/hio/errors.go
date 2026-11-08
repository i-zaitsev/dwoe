// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"errors"
	"net/http"
	"strings"

	"github.com/i-zaitsev/dwoe/pkg/hio/data"
)

const serverErrorMessage = "the server encountered a problem and could not process your request"

func (app *App) logError(r *http.Request, err error) {
	app.Log().Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())
}

func isAPI(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/")
}

func isMissing(err error) bool {
	return errors.Is(err, data.ErrRecordNotFound) || errors.Is(err, data.ErrUnavailable)
}
