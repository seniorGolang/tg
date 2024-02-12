default: transport client swagger

.PHONY: transport
transport:
	go run cmd/tg/main.go transport --services ./example/interfaces --out ./example/gen/transport

.PHONY: client
client:
	go run cmd/tg/main.go client --services ./example/interfaces --outPath ./example/gen/client

.PHONY: swagger
swagger:
	go run cmd/tg/main.go swagger --services ./example/interfaces --outFile ./example/gen/swagger.yml

.PHONY: alias
alias:
	go run cmd/tg/main.go client --services ./example/alias/interfaces --outPath ./example/gen/alias
