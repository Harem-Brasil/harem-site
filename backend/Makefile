.PHONY: all build test bdd test-coverage clean run migrate serve lint

# Default environment
export PORT ?= 40080
export DATABASE_URL ?= postgres://postgres:postgres@localhost:54181/harem?sslmode=disable
export REDIS_URL ?= redis://localhost:6379/0
export JWT_SECRET ?= development-secret-change-in-production

BINARY = harem-api
MAIN = ./cmd/api

all: build

build:
	go build -o $(BINARY) $(MAIN)

test:
	go test -v ./...

bdd:
	go test -v -timeout 5m ./features/...

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

# Docker shortcuts
docker-up:
	docker run -d --name postgres-harem -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=harem -p 54181:5432 postgres:15-alpine || true

docker-down:
	docker stop postgres-harem || true
	docker rm postgres-harem || true

.DEFAULT_GOAL := build
