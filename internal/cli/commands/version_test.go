// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"bytes"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/version"
	"github.com/i-zaitsev/pkg/assert"
)

func TestVersionCmd_Run(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	e := cli.NewEnv(&stdout, &stderr)
	cmd := &cmdVersion{}

	assert.ErrVal(t, cmd.Run(e)).Nil().Fatal()

	want := version.Get() + "\n"
	assert.Eq(t, stdout.String(), want)
}
