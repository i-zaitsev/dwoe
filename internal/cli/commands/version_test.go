// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"bytes"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/version"
)

func TestVersionCmd_Run(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	e := cli.NewEnv(&stdout, &stderr)
	cmd := &cmdVersion{}

	err := cmd.Run(e)
	if err != nil {
		t.Fatal(err)
	}

	want := version.Get() + "\n"
	if got := stdout.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}
