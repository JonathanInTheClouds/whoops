VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"
BINARY  := whoops
CMD     := ./cmd/whoops

.PHONY: build run test clean release install

## Build the binary for your current platform
build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

## Run directly without building a binary
run:
	go run $(LDFLAGS) $(CMD)

## Run all tests
test:
	go test -v -race ./...

## Install the binary to /usr/local/bin
install: build
	mv $(BINARY) /usr/local/bin/$(BINARY)
	@echo "✓ Installed to /usr/local/bin/$(BINARY)"

## Build release binaries for all platforms into dist/
release:
	@mkdir -p dist
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64  $(CMD)
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64  $(CMD)
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64   $(CMD)
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64   $(CMD)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe $(CMD)
	@echo "✓ Binaries built in dist/"

## Remove built binaries
clean:
	rm -f $(BINARY)
	rm -rf dist/

## Show this help
help:
	@echo ""
	@echo "  $(BINARY) $(VERSION)"
	@echo ""
	@grep -E '^##' Makefile | sed 's/## /  /'
	@echo ""