APP_NAME=cardapio-henry-api

.PHONY: run test tidy db-up db-down build

run:
	go run ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy

build:
	go build -o bin/$(APP_NAME) ./cmd/api

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
