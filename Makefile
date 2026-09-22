SHELL := /bin/sh
CLI_VERSION ?= dev
DIST_DIR ?= dist/plugin

.PHONY: all build test test-cli test-plugin test-integration lint verify plugin-dist

all: verify

build:
	@mkdir -p bin
	cd cli && go build -ldflags '-X main.version=$(CLI_VERSION)' -o ../bin/archcore .

test: test-cli test-plugin

test-cli:
	cd cli && go test ./...

test-plugin:
	$(MAKE) -C plugin test

test-integration: build
	$(MAKE) -C plugin test-integration ARCHCORE_BIN=$(CURDIR)/bin/archcore

lint:
	cd cli && go vet ./... && golangci-lint run ./...
	$(MAKE) -C plugin lint

verify: lint test
	$(MAKE) -C plugin check-json check-perms

plugin-dist:
	./scripts/export-plugin.sh "$(DIST_DIR)"
