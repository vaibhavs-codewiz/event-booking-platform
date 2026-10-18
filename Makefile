.PHONY: help build run worker test test-race vet compose-up compose-down compose-logs tidy sqlc fmt

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' Makefile | awk 'BEGIN{FS=":.*##"}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build API and worker binaries into ./bin
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

run: ## Run the API server locally
	go run ./cmd/api

worker: ## Run the background worker locally
	go run ./cmd/worker

test: ## Run all tests
	go test ./...

test-race: ## Run all tests with the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format Go sources
	gofmt -s -w .

compose-up: ## Start PostgreSQL and Redis
	docker compose up -d

compose-down: ## Stop PostgreSQL and Redis (keeps data)
	docker compose down

compose-logs: ## Tail service logs
	docker compose logs -f

tidy: ## Tidy and verify module dependencies
	go mod tidy
	go mod verify

sqlc: ## Generate database access code (requires sqlc)
	sqlc generate
