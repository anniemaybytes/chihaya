STORED_VERSION=$(shell cat VERSION 2>/dev/null)
ifneq ($(STORED_VERSION),)
	VERSION ?= $(STORED_VERSION)
else
	VERSION ?= $(shell git describe --tags --always | sed 's/-/+/' | sed 's/^v//')
endif

BUILDTIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

GOLDFLAGS += -X main.BuildDate=$(BUILDTIME)
GOLDFLAGS += -X main.BuildVersion=$(VERSION)
GOFLAGS = -ldflags "$(GOLDFLAGS)"

DEST := ./bin

.PHONY: all
all: clean build

.PHONY: clean
clean:
	go clean -i ./...
	rm -rf $(DEST)

.PHONY: build
build:
	mkdir -p $(DEST)
	go build -o $(DEST) $(GOFLAGS) ./cmd/...
	strip $(DEST)/*
