// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package batchinfo provides functions for collecting results of a batch work.
// The Result data type holds information about a single collected entry.
// The package does not manage tasks but only collects the information and does reporting.
package batchinfo

import (
	"sync"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/workspace"
)

// Result holds the outcome of processing a single batch entry.
type Result struct {
	Branch  string // branch name
	N       int    // the number of commits or branches created by a worker
	Err     error  // error if the worker failed
	Skipped bool   // true when the workspace could not be resolved
}

// Collect resolves and processes all batch entries in parallel.
// The work function is called for each successfully resolved workspace.
func Collect(
	e *cli.Env,
	rec *batch.Record,
	work func(*workspace.Workspace, batch.Entry) (int, error),
) ([]Result, error) {
	manager, err := e.Manager()
	if err != nil {
		return nil, err
	}
	results := make([]Result, len(rec.Entries))
	var wg sync.WaitGroup
	for index, entry := range rec.Entries {
		wg.Go(func() {
			var res Result
			res.Branch = entry.Branch
			ws, errM := manager.ResolveCompleted(entry.WorkspaceID)
			if errM != nil {
				res.Skipped = true
				res.Err = errM
			} else {
				res.N, res.Err = work(ws, entry)
			}
			results[index] = res
		})
	}
	wg.Wait()
	return results, nil
}

// Report prints per-entry results and returns ok/failed counts.
// The unit is the label for a successful result, e.g. "commit(s)" or "patch(es)".
func Report(e *cli.Env, results []Result, unit string) (ok, failed int) {
	for _, res := range results {
		switch {
		case res.Skipped:
			e.Print("\t%-24s skipped (%v)\n", res.Branch, res.Err)
			failed++
		case res.Err != nil:
			e.Print("\t%-24s failed (%v)\n", res.Branch, res.Err)
			failed++
		default:
			e.Print("\t%-24s %d %s\n", res.Branch, res.N, unit)
			ok++
		}
	}
	return
}
