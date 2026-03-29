Add a `cmd` package with:

- `cmd.go` — parse `-port` flag (default 8080), print
  "Listening on :<port>"
- `cmd_test.go` — test flag parsing

Run `go test ./...` and commit.
