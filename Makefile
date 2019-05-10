TARGETS ?= darwin/amd64 linux/amd64

GO ?= go
TESTS := ./...
TESTFLAGS :=
LDFLAGS := -w -s
GOFLAGS :=
BINDIR := $(CURDIR)/bin

SHELL=/usr/bin/env bash

.PHONY: all
all: build

.PHONY: clean
clean:
	go clean
	rm -rf ./build

.PHONY: build
build:
	GOBIN=$(BINDIR) $(GO) build -o $@/codecommit $(GOFLAGS) -ldflags '$(LDFLAGS)' ./cmd/codecommit

# usage: make clean build-cross dist VERSION=v2.0.0-alpha.3
.PHONY: build-cross
build-cross: LDFLAGS += -extldflags "-static"
build-cross:
	CGO_ENABLED=0 gox -parallel=3 -output="build/_dist/{{.OS}}-{{.Arch}}/{{.Dir}}" -osarch='$(TARGETS)' $(GOFLAGS) $(if $(TAGS),-tags '$(TAGS)',) -ldflags '$(LDFLAGS)' ./cmd/codecommit

.PHONY: test
test: build
test: TESTFLAGS += -race -v -timeout 20m
test: test-unit
test: test-e2e

.PHONY: test-e2e
test-e2e:
	@echo
	@echo "==> Running e2e tests <=="
	$(GO) test $(GOFLAGS) $(TESTFLAGS) ./tests

.PHONY: test-unit
test-unit:
	@echo
	@echo "==> Running unit tests <=="
	$(GO) test $(GOFLAGS) ./pkg/codecommit
