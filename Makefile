VERSION := $(shell git describe --tags --abbrev=0)
BUILDTIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

GOLDFLAGS += -X main.Version=$(VERSION)
GOLDFLAGS += -X main.BuildStamp=$(BUILDTIME)
GOFLAGS = -ldflags "$(GOLDFLAGS)"

.PHONY: run
run: install
	go run ./cmd/tg

.PHONY: install
install:
	go install $(GOFLAGS) ./cmd/tg

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o build/tg ./cmd/tg