// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package docker

import (
	"bytes"
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestParseBuildOutput(t *testing.T) {
	t.Parallel()
	input := `{"stream":"Step 1/3 : FROM alpine\n"}
{"stream":"Step 2/3 : RUN echo hello\n"}
{"stream":"Successfully built abc123\n"}
`
	var buf bytes.Buffer
	assert.NotErr(t, parseBuildOutput(strings.NewReader(input), &buf))
	assert.ContainsAll(t, buf.String(), "Step 1/3", "Step 2/3", "Successfully built")
}

func TestParseBuildOutput_Error(t *testing.T) {
	t.Parallel()
	input := `{"stream":"Step 1/1 : FROM nonexistent\n"}
{"error":"pull access denied for nonexistent"}
`
	var buf bytes.Buffer
	err := parseBuildOutput(strings.NewReader(input), &buf)
	assert.ErrAs[*BuildError](t, err)
}
