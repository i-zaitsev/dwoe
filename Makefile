BINARY_NAME=dwoe
GO=go

.PHONY: all build test lint fmt vet clean dev

all: lint test build

build:
	$(GO) build -o bin/$(BINARY_NAME) ./cmd/dwoe

test:
	$(GO) test ./...

test-v:
	$(GO) test -v ./...

lint: fmt vet ci

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

ci:
	golangci-lint run

clean:
	rm -f $(BINARY_NAME)

dev:
	air

integration-test:
	$(GO) test -v -count=1 -timeout=120s -tags=integration ./test/integration/...
