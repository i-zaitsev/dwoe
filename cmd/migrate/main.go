package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/i-zaitsev/dwoe/internal/migrate"
	"github.com/i-zaitsev/dwoe/internal/util"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: dwoe-migrate <basedir>\n")
		return 1
	}

	path := filepath.Join(os.Args[1], "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	out, count, err := migrate.State(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if count == 0 {
		fmt.Println("already up to date")
		return 0
	}

	if err := util.WriteFileAtomic(path, out, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Printf("migrated %s (%d migration(s) applied)\n", path, count)
	return 0
}
