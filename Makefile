.PHONY: bootstrap db-up db-down migrate dev-api dev-worker dev-web collect generate generate-check lint test test-integration build ci

# Local settings (copy .env.example to .env). Optional: CI sets variables directly.
-include .env
export

GO_DIRS := cmd internal migrations
SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

bootstrap:
	pnpm install --frozen-lockfile
	go mod download

# Local PostgreSQL with PostGIS (compose.yaml).
db-up:
	docker compose up -d --wait

db-down:
	docker compose down

migrate:
	go run ./cmd/lankaweather migrate

# Run the API, the worker and the web dev server in separate terminals.
dev-api: migrate
	go run ./cmd/lankaweather serve

dev-worker: migrate
	go run ./cmd/lankaweather worker

dev-web:
	pnpm dev

# Run every collector once, for example to fill a fresh database.
collect: migrate
	go run ./cmd/lankaweather collect

# Regenerate internal/store from migrations and queries.
generate:
	$(SQLC) generate

generate-check: generate
	git diff --exit-code -- internal/store

lint:
	test -z "$$(gofmt -l $(GO_DIRS))" || (gofmt -l $(GO_DIRS) && exit 1)
	go vet ./...
	go vet -tags=integration ./...
	pnpm format:check
	pnpm lint
	pnpm typecheck

test:
	go test ./...
	pnpm test

# Needs a disposable database: LANKAWEATHER_TEST_DATABASE_URL.
test-integration:
	test -n "$(LANKAWEATHER_TEST_DATABASE_URL)" || (echo "LANKAWEATHER_TEST_DATABASE_URL is not set" && exit 1)
	go test -p 1 -count=1 -tags=integration ./...

build:
	go build -o bin/lankaweather ./cmd/lankaweather
	pnpm build

# Everything CI checks on a pull request. Integration tests run when
# LANKAWEATHER_TEST_DATABASE_URL is set.
ci: lint generate-check test build
	@if [ -n "$(LANKAWEATHER_TEST_DATABASE_URL)" ]; then $(MAKE) test-integration; else echo "LANKAWEATHER_TEST_DATABASE_URL unset: integration tests skipped"; fi
