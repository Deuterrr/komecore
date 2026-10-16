# ==============================================================================
# komecore — Developer Makefile
# ==============================================================================

.PHONY: all build version run dev test test-integration test-contract lint-api check migrate migrate-rollback migrate-sqlserver migrate-sqlserver-rollback docker-up docker-down docker-logs clean

VERSION ?= v1.0.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
LDFLAGS := -w -s -X 'komecore/pkg/version.Version=$(VERSION)' -X 'komecore/pkg/version.GitCommit=$(COMMIT)' -X 'komecore/pkg/version.BuildDate=$(BUILD_DATE)'

all: build

## build: Build production server binary
build:
	go build -ldflags="$(LDFLAGS)" -o bin/komecore ./cmd/komecore

## version: Print binary version
version:
	go run ./cmd/komecore --version

## run: Run server locally
run:
	go run ./cmd/komecore

## dev: Run server with live reload (Air)
dev:
	air

## test: Run full test suite with 0 caching
test:
	go test -v -count=1 ./...

## test-integration: Run ephemeral PostgreSQL 17 & Redis 7 integration test suite
test-integration:
	go test -v -count=1 -tags=integration ./test/integration/...

## test-contract: Run OpenAPI contract and golden snapshot test suite
test-contract:
	go test -v -count=1 ./test/contract/...

## lint-api: Run Spectral OpenAPI quality & style linter
lint-api:
	npx --yes @stoplight/spectral-cli lint api/openapi.yaml --ruleset .spectral.yaml

## check: Run full pre-commit verification (format, vet, tests with race detector, api lint, contract tests)
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

## migrate: Apply database migrations (default: PostgreSQL)
migrate:
	migrate -path ./migrations/postgres -database "$${POSTGRES_DSN:-postgres://komecore:komecore_secret@localhost:5432/komecore_db?sslmode=disable}" up

## migrate-rollback: Rollback 1 migration step (default: PostgreSQL)
migrate-rollback:
	migrate -path ./migrations/postgres -database "$${POSTGRES_DSN:-postgres://komecore:komecore_secret@localhost:5432/komecore_db?sslmode=disable}" down 1

## migrate-sqlserver: Apply database migrations to SQL Server
migrate-sqlserver:
	migrate -path ./migrations/sqlserver -database "$${SQLSERVER_DSN:-sqlserver://sa:KomeCore2026!@localhost:1433?database=komecore_db}" up

## migrate-sqlserver-rollback: Rollback 1 migration step on SQL Server
migrate-sqlserver-rollback:
	migrate -path ./migrations/sqlserver -database "$${SQLSERVER_DSN:-sqlserver://sa:KomeCore2026!@localhost:1433?database=komecore_db}" down 1

## docker-up: Start container stack in background
docker-up:
	docker compose up --build -d

## docker-down: Stop and remove container stack
docker-down:
	docker compose down

## docker-logs: Follow logs from container stack
docker-logs:
	docker compose logs -f

## clean: Remove build artifacts and temporary binaries
clean:
	go clean
	rm -rf bin/
