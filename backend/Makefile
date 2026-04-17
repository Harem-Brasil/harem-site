.PHONY: all build test bdd test-coverage clean run migrate serve lint compose-up compose-up-infra compose-down compose-logs compose-rebuild docker-up docker-down

# Default environment
export PORT ?= 40080
export DATABASE_URL ?= postgres://harem:harem@localhost:5432/harem?sslmode=disable
export REDIS_URL ?= redis://localhost:6379/0
export JWT_SECRET ?= development-secret-change-in-production

BINARY = harem-api
MAIN = ./cmd/api

all: build

build:
	go build -o $(BINARY) $(MAIN)

test:
	go test -v ./...

bdd: migrate
	go test -v -race -timeout 5m ./features/...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	go fmt ./...
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html

run: build
	./$(BINARY) serve

migrate: build
	./$(BINARY) migrate

serve: build
	./$(BINARY) serve

dev:
	go run $(MAIN) serve

dev-migrate:
	go run $(MAIN) migrate

# Docker Compose shortcuts
compose-up:
	docker compose up -d

compose-up-infra:
	docker compose up -d postgres redis

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f api

compose-rebuild:
	docker compose build --no-cache api
	docker compose up -d api

# Legacy: single-container shortcuts (use compose-up instead)
docker-up: compose-up-infra

docker-down: compose-down

.DEFAULT_GOAL := build
