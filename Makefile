.PHONY: build clean install test

BINARY_NAME=codex-market
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION_PKG=github.com/egoavara/codex-market/internal/version
LDFLAGS=-ldflags "-s -w -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).BuildDate=$(BUILD_DATE)"

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*

install: build
	cp $(BINARY_NAME) $(HOME)/.local/bin/

test:
	go test -v ./...

# Cross-compilation
build-all: build-darwin build-linux build-windows

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .
