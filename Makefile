BINARY_NAME=dwoe
GO=go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -X github.com/i-zaitsev/dwoe/internal/version.version=$(VERSION)

.PHONY: all build install test lint fmt vet clean dev
.PHONY: images image-base image-go image-python image-c image-cpp image-universal image-proxy

all: lint test build

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/dwoe

build-migrate:
	$(GO) build -o bin/dwoe-migrate ./cmd/migrate

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

# Agent images.
# Every image is built from dwoe-base and carries both agent CLIs, so any image can run any provider.
DOCKER_BUILD = docker build -f docker/Dockerfile

image-base:
	$(DOCKER_BUILD).base -t dwoe-base:latest docker/

image-go: image-base
	$(DOCKER_BUILD).golang -t dwoe-agent:go docker/

image-python: image-base
	$(DOCKER_BUILD).python -t dwoe-agent:python docker/

image-c: image-base
	$(DOCKER_BUILD).c -t dwoe-agent:c docker/

image-cpp: image-base
	$(DOCKER_BUILD).cpp -t dwoe-agent:cpp docker/

image-universal: image-base
	$(DOCKER_BUILD).universal -t dwoe-agent:latest docker/

image-proxy:
	$(DOCKER_BUILD).proxy -t dwoe-proxy:latest docker/

images: image-universal image-go image-python image-c image-cpp image-proxy
