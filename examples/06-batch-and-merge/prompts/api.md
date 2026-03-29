Add a `server` package with:

- `server.go` — a simple HTTP handler that responds with
  `{"status": "ok"}` on GET /health
- `server_test.go` — test using `httptest`

Run `go test ./...` and commit.
