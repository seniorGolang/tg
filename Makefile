VERSION := $(shell git describe --tags --abbrev=0)
BUILDTIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

GOLDFLAGS += -X main.Version=$(VERSION)
GOLDFLAGS += -X main.BuildStamp=$(BUILDTIME)
GOFLAGS = -ldflags "$(GOLDFLAGS)"

run: install
	./mybinary

install:
	go install $(GOFLAGS) ./cmd/tg

.PHONY: generate
generate: transport client swagger

.PHONY: transport
transport:
	go run cmd/tg/main.go transport --services ./example/interfaces --out ./example/transport

.PHONY: client
client:
	rm -rf example/clients
	go run cmd/tg/main.go client --services ./example/interfaces --outPath ./example/clients/example --go

.PHONY: swagger
swagger:
	go run cmd/tg/main.go swagger --services ./example/interfaces --outFile ./example/swagger.yaml
