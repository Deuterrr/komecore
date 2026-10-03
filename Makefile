# ==============================================================================
# komecore — Developer Makefile
# ==============================================================================

.PHONY: all build run dev test test-integration check migrate migrate-rollback migrate-supabase migrate-supabase-all migrate-sqlserver migrate-sqlserver-rollback seed seed-supabase docker-up docker-down docker-logs clean


all: build

## build: Build production server binary
build:
	go build -ldflags="-w -s" -o bin/komecore ./cmd/komecore

## run: Run server locally
run:
	go run ./cmd/komecore

## dev: Run server with live reload (Air)
dev:
	air

## test: Run full test suite with 0 caching
test:
	go test -v -count=1 ./...

## test-integration: Run ephemeral PostgreSQL integration test suite
test-integration:
	go test -v -count=1 -tags=integration ./test/integration/...

## check: Run full pre-commit verification (format, vet, tests with race detector)
check:
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l internal cmd pkg seeds tools test)" || (echo "Unformatted files found:" && gofmt -l internal cmd pkg seeds tools test && exit 1)
	@echo "Running go vet..."
	go vet ./...
	@echo "Running tests with race detector..."
	go test -race -count=1 ./...
	@echo "All quality checks passed!"

## migrate: Apply database migrations (default: PostgreSQL)
migrate:
	go run ./cmd/migrate -target=postgres

## migrate-rollback: Rollback 1 migration step (default: PostgreSQL)
migrate-rollback:
	go run ./cmd/migrate -target=postgres -rollback=true

## migrate-supabase: Apply database migrations to Supabase (DB only)
migrate-supabase:
	go run ./cmd/migrate -target=supabase

## migrate-supabase-all: Apply migrations and storage buckets to Supabase
migrate-supabase-all:
	go run ./cmd/migrate -target=supabase -storage=true

## migrate-sqlserver: Apply database migrations to SQL Server
migrate-sqlserver:
	go run ./cmd/migrate -target=sqlserver

## migrate-sqlserver-rollback: Rollback 1 migration step on SQL Server
migrate-sqlserver-rollback:
	go run ./cmd/migrate -target=sqlserver -rollback=true


## seed: Seed initial database fixtures (default: PostgreSQL)
seed:
	go run ./cmd/seed -target=postgres

## seed-supabase: Seed initial database fixtures to Supabase
seed-supabase:
	go run ./cmd/seed -target=supabase

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
