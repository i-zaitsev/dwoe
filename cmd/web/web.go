// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"os"

	"github.com/i-zaitsev/dwoe/web"
)

func main() {
	os.Exit(web.Main(os.Args[1:]))
}
