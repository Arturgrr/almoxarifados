.PHONY: db migrate sqlc openapi dev test

db:
	docker compose up -d

migrate:
	go run ./cmd/terndotenv $(ARGS)

sqlc:
	sqlc generate

openapi:
	go run ./cmd/openapi

dev:
	go tool air

test:
	go test ./...
