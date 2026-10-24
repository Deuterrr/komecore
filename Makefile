# ==============================================================================
# komecore — Developer Makefile
# ==============================================================================

.PHONY: all build version run dev test test-integration test-contract lint-api check migrate migrate-rollback migrate-version migrate-sqlserver migrate-sqlserver-rollback migrate-sqlserver-version seed seed-dev docker-up docker-down docker-logs clean

VERSION ?= v1.0.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
LDFLAGS := -w -s -X 'komecore/pkg/version.Version=$(VERSION)' -X 'komecore/pkg/version.GitCommit=$(COMMIT)' -X 'komecore/pkg/version.BuildDate=$(BUILD_DATE)'

all: build

## Build production server binary
build:
	go build -ldflags="$(LDFLAGS)" -o bin/komecore ./cmd/komecore

version:
	go run ./cmd/komecore --version

## Run server locally
run:
	go run ./cmd/komecore
## Run server with live reload
dev:
	air

## Run full test suite with 0 caching
test:
	go test -v -count=1 ./...
## Run ephemeral PostgreSQL 17 & Redis 7 integration test suite
test-integration:
	go test -v -count=1 -tags=integration ./test/integration/...
## Run OpenAPI contract and golden snapshot test suite
test-contract:
	go test -v -count=1 ./test/contract/...

## Run Spectral OpenAPI quality & style linter
lint-api:
	npx --yes @stoplight/spectral-cli lint api/openapi.yaml --ruleset .spectral.yaml

## Run full pre-commit verification (format, vet, tests with race detector, api lint, contract tests)
check:
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l internal cmd pkg test)" || (echo "Unformatted files found:" && gofmt -l internal cmd pkg test && exit 1)
	@echo "Running go vet..."
	go vet ./...
	@echo "Linting OpenAPI specification..."
	npx --yes @stoplight/spectral-cli lint api/openapi.yaml --ruleset .spectral.yaml
	@echo "Running tests with race detector..."
	go test -race -count=1 ./...
	@echo "Running contract tests..."
	go test -v -count=1 ./test/contract/...
	@echo "All quality checks passed!"

## Apply database migrations (default: PostgreSQL)
migrate:
	go run ./cmd/komecore --migrate
## Rollback 1 migration step (default: PostgreSQL)
migrate-rollback:
	go run ./cmd/komecore --migrate-rollback
## Print current applied database migration version (PostgreSQL)
migrate-version:
	go run ./cmd/komecore --migrate-version

## Apply database migrations to Supabase
migrate-supabase:
	go run ./cmd/komecore --target=supabase --migrate
## Rollback 1 migration step on Supabase
migrate-supabase-rollback:
	go run ./cmd/komecore --target=supabase --migrate-rollback
## Print current applied migration version (Supabase)
migrate-supabase-version:
	go run ./cmd/komecore --target=supabase --migrate-version

## Populate base/production database seeds
seed:
	go run ./cmd/komecore --seed
## Populate development seeds (e.g. shops, products, customers)
seed-dev:
	go run ./cmd/komecore --seed-dev

## Start container stack in background
docker-up:
	docker compose up --build -d
## Stop and remove container stack
docker-down:
	docker compose down
## Follow logs from container stack
docker-logs:
	docker compose logs -f

## Remove build artifacts and temporary binaries
clean:
	go clean
	rm -rf bin/
