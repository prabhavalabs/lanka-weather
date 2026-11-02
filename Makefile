.PHONY: bootstrap dev-api dev-web lint test build ci

GO_DIRS := cmd internal

bootstrap:
	pnpm install --frozen-lockfile
	go mod download

# Run the API and the web dev server in two terminals.
dev-api:
	go run ./cmd/openweather serve

dev-web:
	pnpm dev

lint:
	test -z "$$(gofmt -l $(GO_DIRS))" || (gofmt -l $(GO_DIRS) && exit 1)
	go vet ./...
	pnpm format:check
	pnpm lint
	pnpm typecheck

test:
	go test ./...
	pnpm test

build:
	go build -o bin/openweather ./cmd/openweather
	pnpm build

# Everything CI checks on a pull request.
ci: lint test build
