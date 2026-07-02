BINARY       := softlayer
DIST         := dist
INSTALL_PATH := /usr/local/bin
VERSION      := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOFLAGS      := -trimpath
LDFLAGS      := -s -w -X main.version=$(VERSION)
PLATFORMS    := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: all build build-all test lint install clean $(PLATFORMS)

all: build

## build: build for the host platform
build:
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY) .

## build-all: cross-compile static binaries for all platforms into dist/
build-all: $(PLATFORMS)

$(PLATFORMS):
	GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) CGO_ENABLED=0 \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
		-o $(DIST)/$(BINARY)-$(subst /,-,$@) .

## test: run unit tests with the race detector
test:
	go test -race ./...

## lint: run golangci-lint
lint:
	golangci-lint run

## install: install the binary to $(INSTALL_PATH)
install: build
	install $(BINARY) $(INSTALL_PATH)

## clean: remove build artifacts
clean:
	rm -rf $(BINARY) $(DIST)
