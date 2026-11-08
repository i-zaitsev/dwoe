// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package hio

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/i-zaitsev/pkg/assert"
)

func TestLayout_NoMarkupInGoFiles(t *testing.T) {
	markup := []string{"hx-", "<div", "<span", "<table", "<p>", "<a href"}

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "ui" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, errRead := os.ReadFile(path)
		assert.ErrVal(t, errRead).Nil().Fatal()
		for _, m := range markup {
			if strings.Contains(string(content), m) {
				t.Errorf("%s contains %q", path, m)
			}
		}
		return nil
	})
	assert.ErrVal(t, err).Nil().Fatal()
}

func TestLayout_HTMXLivesInLayoutAndPartials(t *testing.T) {
	pages, err := filepath.Glob(filepath.Join("ui", "templates", "pages", "*.html"))
	assert.ErrVal(t, err).Nil().Fatal()
	assert.True(t, len(pages) > 0)

	for _, page := range pages {
		content, errRead := os.ReadFile(page)
		assert.ErrVal(t, errRead).Nil().Fatal()
		for _, attr := range []string{"hx-", "sse-"} {
			if strings.Contains(string(content), attr) {
				t.Errorf("%s contains %q: move it to a partial", page, attr)
			}
		}
	}
}
