GO := go
GOFLAGS := -mod=vendor
BINDIR := $(CURDIR)/bin

COMMIT := $(shell git rev-parse HEAD)
IMAGE_TAG:=$(shell ./docker/image-tag)

LDFLAGS := -w -s -X main.Version=$(IMAGE_TAG) -X main.GitCommit="$(COMMIT)"

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
test: TESTFLAGS += -race -v -timeout 20m
test: build test-unit test-e2e

.PHONY: test-e2e
test-e2e:
	@echo
	@echo "==> Running e2e tests DISABLED <=="
	#$(GO) test $(GOFLAGS) $(TESTFLAGS) ./tests

.PHONY: test-unit
test-unit:
	@echo
	@echo "==> Running unit tests <=="
	$(GO) test $(GOFLAGS) ./pkg/codecommit

.PHONY: build-docker
build-docker:
	@echo
	@echo "==> Build Docker Image DISABLED <=="
	#mkdir -p build
	#rm -rf build/docker
	#cp -a docker build/.
	#find . -maxdepth 1 ! -regex './build\|\.' -print0 | xargs -0 -l1 -I{} cp -a {} build/docker/.
	#docker build \
	#	-t docker.io/bashims/go-codecommit:$(IMAGE_TAG) \
	#	--build-arg=GO_CODECOMMIT_VER=$(IMAGE_TAG) \
	#	--build-arg=GO_CODECOMMIT_COMMIT="$(COMMIT)" \
	#	build/docker/.
.PHONY: push-docker
push-docker:
	@echo
	@echo "==> Push Docker Image DISABLED <=="
	#docker push docker.io/bashims/go-codecommit:$(IMAGE_TAG)