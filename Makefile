BINARY_NAME=dwoe
GO=go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -X github.com/i-zaitsev/dwoe/internal/version.version=$(VERSION)

.PHONY: all build install test lint fmt vet clean dev

all: lint test build

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/dwoe

install: build
	sudo mv bin/$(BINARY_NAME) /usr/local/bin/

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
