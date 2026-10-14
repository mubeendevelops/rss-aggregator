# Variables are read from $(ENV_FILE). Use `make ENV_FILE=.env.example <target>`
# to run against the docker-compose defaults without a local .env.
ENV_FILE ?= .env
-include $(ENV_FILE)
export

.PHONY: db-up db-down migrate migrate-down sqlc run build lint vuln smoke

db-up:
	docker compose up -d --wait

db-down:
	docker compose down

migrate:
	@goose -dir sql/schema postgres "$(DB_URL)" up

migrate-down:
	@goose -dir sql/schema postgres "$(DB_URL)" down

sqlc:
	sqlc generate

run:
	go run ./cmd/rss-aggregator

build:
	go build -o bin/rss-aggregator ./cmd/rss-aggregator

lint:
	golangci-lint run ./...

vuln:
	govulncheck ./...

smoke:
	./scripts/smoke.sh
