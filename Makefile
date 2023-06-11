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
all: clean chihaya cc bencode

.PHONY: clean
clean:
	go clean -i ./...
	rm -rf $(DEST)

.PHONY: chihaya
chihaya:
	mkdir -p $(DEST)
	go build -pgo=auto -o $(DEST) $(GOFLAGS) ./cmd/chihaya
	strip $(DEST)/chihaya

.PHONY: cc
cc:
	mkdir -p $(DEST)
	go build -o $(DEST) $(GOFLAGS) ./cmd/cc
	strip $(DEST)/cc

.PHONY: bencode
bencode:
	mkdir -p $(DEST)
	go build -o $(DEST) $(GOFLAGS) ./cmd/bencode
	strip $(DEST)/bencode
