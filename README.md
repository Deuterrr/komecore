# komecore

**A high-performance, modular general e-commerce engine written in Go.**

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![Router](https://img.shields.io/badge/Router-Chi%20v5-blue)](https://github.com/go-chi/chi)
[![Database](https://img.shields.io/badge/Database-PostgreSQL%2016+-4169E1?logo=postgresql)](https://www.postgresql.org)
[![Payments](https://img.shields.io/badge/Payments-Midtrans%20Core%20API-002f6c)](https://midtrans.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

### Overview

**komecore** is a clean, scalable backend service engineered for modern e-commerce platforms. Built with **Clean Architecture** and **Domain-Driven Design (DDD)** principles, it provides a lean, framework-agnostic core capable of managing complete commerce lifecycles—from authentication and multi-shop catalog management to checkout, dynamic pricing, automated payment reconciliation, and fulfillment.

### License

This project is open-source software licensed under the [MIT License](LICENSE).

---

### Quick Start

> Configure `DB_TARGET` (`postgres` · `supabase` · `sqlserver`) and `STORAGE_PROVIDER` (`supabase` · `gcs` · `noop`) in `.env`. Migration SQL lives in `migrations/<dialect>/`.

#### 1. Docker Compose (Fastest Stack)

```bash
docker compose up --build -d
curl http://localhost:7129/health
```

* **Tools & Environment Provided**: PostgreSQL 16 database container, isolated app server container, Docker network.

#### 2. Live Reload / Hot Reload (`air`)

```bash
cp .env.example .env
# set DB_TARGET=postgres | supabase | sqlserver in .env
make migrate && make seed
make dev   # or run `air`
```

* **Tools & Environment Provided**: Air live-reload daemon (`.air.toml`), automated instant rebuilds on `.go` file save *(compatible with alternative watchers like `CompileDaemon` or `modd`)*.

#### 3. Makefile Automation

```bash
cp .env.example .env
make migrate        # or: make migrate-supabase / make migrate-sqlserver
make seed
make run
make test
```

* **Tools & Environment Provided**: Task runner CLI (`Makefile`), custom schema migration tool (`cmd/migrate`), database seeder (`cmd/seed`).

#### 4. Traditional Go Toolchain

```bash
cp .env.example .env
go mod download
go run ./cmd/migrate -target=postgres   # or -target=supabase / -target=sqlserver
go run ./cmd/seed
go run ./cmd/komecore
go test -v -count=1 ./...
```

* **Tools & Environment Provided**: Direct Go toolchain (`go` CLI), manual script execution, source UTF-8 BOM cleaner (`tools/utf8_bom_cleaner.go`).

#### Compatible Development Tools

This repository is pre-configured and compatible with standard developer tooling:
* **Live Reloading**: Pre-configured for [Air](https://github.com/air-verse/air) (`.air.toml` / `make dev`), and compatible with any file watcher (**CompileDaemon**, **Modd**, or **watchexec**).
* **Containers & Runtimes**: Docker Desktop, Docker Compose, or Podman for service isolation.
* **IDEs & Code Editors**: VS Code (with Go extension), GoLand, or Neovim / LSP.
* **Database Management**: `psql` CLI, DBeaver, TablePlus, or pgAdmin for PostgreSQL 16.
* **API Testing & Client**: Postman, Bruno, Insomnia, or `curl` for testing API endpoints.
* **Linters & Formatters**: `golangci-lint`, `go vet`, `gofmt`, and internal repository scripts (`tools/utf8_bom_cleaner.go`).

### API Documentation & Testing Collections

Complete, pre-configured API collections, specs, and environment files are located in the [`api/`](api/) directory:

* **Postman**: Import [`api/postman_collection.json`](api/postman_collection.json) directly into Postman Desktop/Web.
* **VS Code REST Client**: Open [`api/komecore.http`](api/komecore.http) and click **Send Request** directly inside your editor.
* **Bruno**: Open the [`api/bruno`](api/bruno) folder in the [Bruno API Client](https://www.usebruno.com/).
* **OpenAPI / Swagger**: View or import [`api/openapi.yaml`](api/openapi.yaml) into Swagger UI, Redoc, or Insomnia.

For step-by-step instructions on setting up environment variables and testing auth flows, see the [API README](api/README.md).

---

### Dependencies & Tech Stack

* **Runtime & Routing**: [Go 1.25+](https://golang.org) with [Chi v5](https://github.com/go-chi/chi) for fast, zero-allocation composable routing.
* **Database & Storage**: [PostgreSQL 16+](https://www.postgresql.org) via [pgx v5](https://github.com/jackc/pgx) with `TIMESTAMPTZ` timezone consistency; [Supabase / S3](https://supabase.com) for media assets.
* **External Integrations**: [Midtrans Core API](https://midtrans.com) (VA, QRIS, e-Wallets), [Google OAuth2](https://developers.google.com/identity), and SMTP for OTP email challenges.
* **Security & Observability**: [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt), [crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto), and [uber-go/zap](https://github.com/uber-go/zap) structured audit logging.

### Features

* **Identity & Access**: Dual-actor auth (Customer and Staff) using JWT & secure HTTP-only cookies, Google OAuth2 social login, role/permission guards, and SMTP-driven OTP verification & password resets.
* **Catalog & Multi-Shop**: Merchant shop isolation, real-time stock reservations and rollbacks, automated multi-resolution image transformation to Supabase/S3, and SEO slug generation.
* **Cart & Checkout Engine**: Flexible product variants stored via PostgreSQL `JSONB`, dynamic real-time price aggregation (subtotals, shipping, gateway fees), and atomic stock validation.
* **Payments & Background Workers**: Midtrans Core API (Virtual Accounts, QRIS, e-Wallets), signature-verified idempotent webhooks, plus autonomous cron jobs for payment reconciliation, past-due expiry, and order SLA refund processing.
* **Fulfillment & Logistics**: Multi-tier courier rate estimation, manual/integrated waybill dispatch, and lifecycle shipment tracking (`pending` → `shipped` → `delivered`).
* **Reliability & Observability**: Sliding-window rate limiting, structured Zap audit logging, and graceful termination that drains active workers before closing database pools.
