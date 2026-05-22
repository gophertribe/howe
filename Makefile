.PHONY: all deps build build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 \
        release release-linux-amd64 release-linux-arm64 release-darwin-amd64 release-darwin-arm64 \
        snapshot snapshot-linux-amd64 snapshot-linux-arm64 snapshot-darwin-amd64 snapshot-darwin-arm64 \
        clean

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS = -s -w -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

all: build

## Dependencies
deps:
	go mod download
	go mod tidy

## Build all targets for the current host OS
build: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64

## Individual platform builds (using go build directly)
build-linux-amd64:
	@mkdir -p release/
	@echo "Building howe linux/amd64 $(VERSION) ($(COMMIT))..."
	@CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="$(LDFLAGS)" -o release/howe-linux-amd64 ./cmd/howe

build-linux-arm64:
	@mkdir -p release/
	@echo "Building howe linux/arm64 $(VERSION) ($(COMMIT))..."
	@CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -ldflags="$(LDFLAGS)" -o release/howe-linux-arm64 ./cmd/howe

build-darwin-amd64:
	@mkdir -p release/
	@echo "Building howe darwin/amd64 $(VERSION) ($(COMMIT))..."
	@CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -ldflags="$(LDFLAGS)" -o release/howe-darwin-amd64 ./cmd/howe

build-darwin-arm64:
	@mkdir -p release/
	@echo "Building howe darwin/arm64 $(VERSION) ($(COMMIT))..."
	@CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -ldflags="$(LDFLAGS)" -o release/howe-darwin-arm64 ./cmd/howe

## Release all platforms using GoReleaser (creates GitHub release on tag)
release:
	goreleaser release --clean

## Release a single platform using GoReleaser
release-linux-amd64:
	goreleaser release --clean --single-target --id howe -p 1 \
		--snapshot \
		--output release/howe-linux-amd64

release-linux-arm64:
	goreleaser release --clean --single-target --id howe -p 1 \
		--snapshot \
		--output release/howe-linux-arm64

release-darwin-amd64:
	goreleaser release --clean --single-target --id howe -p 1 \
		--snapshot \
		--output release/howe-darwin-amd64

release-darwin-arm64:
	goreleaser release --clean --single-target --id howe -p 1 \
		--snapshot \
		--output release/howe-darwin-arm64

## Snapshot builds (no release, for local testing)
snapshot:
	goreleaser release --clean --snapshot

snapshot-linux-amd64:
	goreleaser build --clean --single-target --id howe -p 1 --snapshot \
		-o release/howe-linux-amd64

snapshot-linux-arm64:
	goreleaser build --clean --single-target --id howe -p 1 --snapshot \
		-o release/howe-linux-arm64

snapshot-darwin-amd64:
	goreleaser build --clean --single-target --id howe -p 1 --snapshot \
		-o release/howe-darwin-amd64

snapshot-darwin-arm64:
	goreleaser build --clean --single-target --id howe -p 1 --snapshot \
		-o release/howe-darwin-arm64

## Clean build artifacts
clean:
	rm -rf release/ dist/
