# Everything you run day to day. `make` with no target lists them.
#
# Make never reads .env itself (it would mangle $, #, and spaces). Targets that
# need it run through scripts/with-env.sh; docker compose reads .env on its own.

ENV     := ./scripts/with-env.sh
COMPOSE ?= docker compose
# The release identifier stamped into builds and shown on every log line
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.DEFAULT_GOAL := help
.PHONY: help setup dev dev-api dev-worker dev-web db-up kc-up tp-up db-migrate db-status db-down db-reset db-shell \
        test test-integration cover lint fmt generate generate-check build up down e2e check reset rename clean

## help: list targets
help:
	@grep -hE '^## ' Makefile | sed -e 's/^## //' | awk '{i = index($$0, ": "); printf "  \033[36m%-17s\033[0m %s\n", substr($$0, 1, i - 1), substr($$0, i + 2)}'

## setup: first run: create .env, download Go modules, install web deps
setup:
	@test -f .env || (cp .env.example .env && echo "created .env")
	go mod download
	cd web && npm ci

## dev: run Postgres, Keycloak, and Temporal, apply migrations, then the API, worker, and web app (Ctrl-C stops all)
dev: db-up kc-up tp-up db-migrate
	@trap 'kill 0' INT TERM EXIT; \
	$(MAKE) --no-print-directory dev-api & \
	$(MAKE) --no-print-directory dev-worker & \
	$(MAKE) --no-print-directory dev-web & \
	wait

## dev-api: run the Go API
dev-api:
	$(ENV) go run ./cmd/api

## dev-worker: run the Temporal worker
dev-worker:
	$(ENV) go run ./cmd/worker

## dev-web: run the Next.js dev server
dev-web:
	$(ENV) sh -c 'cd web && exec npx next dev -p "$${WEB_PORT:-3000}"'

## db-up: start Postgres and wait until it is healthy
db-up:
	$(COMPOSE) up -d --wait db

## kc-up: start Keycloak (http://localhost:8180, admin/admin) and wait until it is ready
kc-up:
	$(COMPOSE) up -d --wait keycloak

## tp-up: start Temporal (UI on http://localhost:8233) and wait until it is ready
tp-up:
	$(COMPOSE) up -d --wait temporal

## db-migrate: apply pending migrations (the API never does this itself)
db-migrate:
	$(ENV) go run ./cmd/api migrate

## db-status: list applied and pending migrations
db-status:
	$(ENV) go run ./cmd/api migrate -status

## db-down: stop Postgres (data is kept)
db-down:
	$(COMPOSE) down

## db-reset: stop Postgres and delete its data
db-reset:
	$(COMPOSE) down -v

## db-shell: open psql in the database container
db-shell:
	$(COMPOSE) exec db sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

## test: Go unit tests with the race detector, and the .env round-trip test
test:
	go test -race ./...
	./scripts/test-env.sh

## test-integration: Go tests against the real database (needs db-up)
test-integration:
	$(ENV) go test -race -tags integration -count=1 ./...

## cover: unit test coverage report
cover:
	go test -coverpkg=./internal/... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

## generate: regenerate the Go server and TypeScript types from api/openapi.yaml
generate:
	go generate ./api/
	cd web && npx openapi-typescript ../api/openapi.yaml -o lib/api/schema.ts

## generate-check: fail if the committed generated code doesn't match the spec
generate-check: generate
	@git diff --exit-code -- api/api.gen.go web/lib/api/schema.ts \
		|| { echo "generated code is out of date with api/openapi.yaml: run make generate and commit"; exit 1; }

## lint: gofmt, go vet, golangci-lint, eslint, tsc, prettier
lint:
	@out=$$(gofmt -l cmd internal api); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	go vet ./...
	@if command -v golangci-lint >/dev/null; then golangci-lint run; else echo "golangci-lint not installed, skipping"; fi
	cd web && npm run lint && npm run typecheck && npm run format:check

## fmt: format Go and web code
fmt:
	gofmt -w cmd internal api
	cd web && npm run format

## build: compile the API and worker to bin/ and build the web app
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-X main.version=$(VERSION)" -o bin/ ./cmd/api ./cmd/worker
	cd web && npm run build

## up: build and run the whole stack in Docker (web on :3000, API on :8080, Temporal UI on :8233)
up:
	$(COMPOSE) --profile full up -d --build --wait

## e2e: browser tests (Playwright) against the running stack; start it with make up
e2e:
	$(ENV) sh -c 'cd web && E2E_BASE_URL="$${E2E_BASE_URL:-http://localhost:$${WEB_PORT:-3000}}" E2E_API_URL="$${E2E_API_URL:-http://localhost:$${PORT:-8080}}" exec npx playwright test'

## down: stop the whole Docker stack
down:
	$(COMPOSE) --profile full down

## check: lint, contract, test, and build, as CI does
check: lint generate-check test build

## reset: remove the todo demo, leaving GET /, /livez, /readyz, /docs, and the landing page
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
