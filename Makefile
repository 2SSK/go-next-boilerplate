# Everything you run day to day. `make` with no target lists them.

-include .env
export

DB_PORT  ?= 5432
POSTGRES_USER ?= app
POSTGRES_DB   ?= app
WEB_PORT ?= 3000
COMPOSE  ?= docker compose

.DEFAULT_GOAL := help
.PHONY: help setup dev dev-api dev-web db-up db-down db-reset db-shell \
        test test-integration cover lint fmt build up down check reset rename clean

## help: list targets
help:
	@grep -hE '^## ' Makefile | sed -e 's/^## //' | awk '{i = index($$0, ": "); printf "  \033[36m%-17s\033[0m %s\n", substr($$0, 1, i - 1), substr($$0, i + 2)}'

## setup: first run: create .env, download Go modules, install web deps
setup:
	@test -f .env || (cp .env.example .env && echo "created .env")
	go mod download
	cd web && npm ci

## dev: run Postgres, the API, and the web app together (Ctrl-C stops all)
dev: db-up
	@trap 'kill 0' INT TERM EXIT; \
	$(MAKE) --no-print-directory dev-api & \
	$(MAKE) --no-print-directory dev-web & \
	wait

## dev-api: run the Go API
dev-api:
	go run ./cmd/api

## dev-web: run the Next.js dev server
dev-web:
	cd web && npx next dev -p $(WEB_PORT)

## db-up: start Postgres and wait until it is healthy
db-up:
	$(COMPOSE) up -d --wait db

## db-down: stop Postgres (data is kept)
db-down:
	$(COMPOSE) down

## db-reset: stop Postgres and delete its data
db-reset:
	$(COMPOSE) down -v

## db-shell: open psql in the database container
db-shell:
	$(COMPOSE) exec db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

## test: Go unit tests with the race detector
test:
	go test -race ./...

## test-integration: Go tests against the real database (needs db-up)
test-integration:
	go test -race -tags integration -count=1 ./...

## cover: unit test coverage report
cover:
	go test -coverpkg=./internal/... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

## lint: gofmt, go vet, golangci-lint, eslint, tsc, prettier
lint:
	@out=$$(gofmt -l cmd internal); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	go vet ./...
	@if command -v golangci-lint >/dev/null; then golangci-lint run; else echo "golangci-lint not installed, skipping"; fi
	cd web && npm run lint && npm run typecheck && npm run format:check

## fmt: format Go and web code
fmt:
	gofmt -w cmd internal
	cd web && npm run format

## build: compile the API to bin/api and build the web app
build:
	CGO_ENABLED=0 go build -trimpath -o bin/api ./cmd/api
	cd web && npm run build

## up: build and run the whole stack in Docker (web on :3000, API on :8080)
up:
	$(COMPOSE) --profile full up -d --build --wait

## down: stop the whole Docker stack
down:
	$(COMPOSE) --profile full down

## check: lint, test, and build, as CI does
check: lint test build

## reset: remove the todo demo, leaving GET /, GET /healthz, and the landing page
reset:
	@if [ "$(YES)" != "1" ]; then \
		printf "Delete the todo demo (Go, SQL, and web files)? [y/N] "; read ans; [ "$$ans" = y ]; \
	fi
	./scripts/reset.sh

## rename: re-brand the project, e.g. make rename NAME=myapp MODULE=github.com/me/myapp
rename:
	@test -n "$(NAME)" -a -n "$(MODULE)" || (echo "usage: make rename NAME=myapp MODULE=github.com/me/myapp"; exit 1)
	./scripts/rename.sh "$(NAME)" "$(MODULE)"

## clean: remove build output
clean:
	rm -rf bin coverage.out web/.next
