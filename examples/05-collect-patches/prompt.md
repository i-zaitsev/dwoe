Add a `utils` package with a `Slugify(s string) string` function that
converts a string to a URL-friendly slug (lowercase, spaces to hyphens,
strip non-alphanumeric characters).

Add tests in `utils/slugify_test.go`. Run `go test ./...` and make
sure everything passes. Commit the result.
