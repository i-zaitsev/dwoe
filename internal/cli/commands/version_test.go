// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"bytes"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/version"
)

func TestVersionCmd_Run(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	e := cli.NewEnv(&stdout, &stderr)
	cmd := &cmdVersion{}

	assert.NotErr(t, cmd.Run(e))

	want := version.Get() + "\n"
	assert.Equal(t, stdout.String(), want)
}
