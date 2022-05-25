VERSION := $(shell git describe --tags --abbrev=0)
BUILDTIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

GOLDFLAGS += -X main.Version=$(VERSION)
GOLDFLAGS += -X main.BuildStamp=$(BUILDTIME)
GOFLAGS = -ldflags "$(GOLDFLAGS)"

run: install
	./mybinary

install:
	go install $(GOFLAGS) ./cmd/tg