.PHONY: release build deps

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS = -s -w -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

none:
	@echo "Please use make (build|deps|release)"

deps:
	go mod download
	go mod tidy

build:
	@mkdir -p release/
	@echo "Building howe $(VERSION) ($(COMMIT))..."
	@GOARCH=amd64 GOOS=linux go build -ldflags="$(LDFLAGS)" -o release/howe-linux-amd64 ./cmd/howe
	@if [ "$$(uname -m)" = "arm64" ]; then \
		echo "Building darwin/arm64 (native architecture)..."; \
		GOARCH=arm64 GOOS=darwin go build -ldflags="$(LDFLAGS)" -o release/howe-darwin-arm64 ./cmd/howe; \
	else \
		echo "Building darwin/amd64..."; \
		GOARCH=amd64 GOOS=darwin go build -ldflags="$(LDFLAGS)" -o release/howe-darwin-amd64 ./cmd/howe; \
	fi

release: build
	@upx --brute release/howe-linux-amd64 -o release/howe-linux-amd64-compressed
	@if [ -f release/howe-darwin-arm64 ]; then \
		upx --brute release/howe-darwin-arm64 -o release/howe-darwin-arm64-compressed; \
	elif [ -f release/howe-darwin-amd64 ]; then \
		upx --brute release/howe-darwin-amd64 -o release/howe-darwin-amd64-compressed; \
	fi
	@shasum release/* > release/shasums
