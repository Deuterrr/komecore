# 🌾KomeCore

Clean, scalable backend service engineered for modern e-commerce platforms in Go.

<img src="assets/placeholder.png" height=300>

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![Router](https://img.shields.io/badge/Router-Chi%20v5-blue)](https://github.com/go-chi/chi)
[![Database](https://img.shields.io/badge/Databases-PostgreSQL%20|%20SQL%20Server-4169E1?logo=postgresql)](https://www.postgresql.org)
[![Payments](https://img.shields.io/badge/Payments-Midtrans%20Core%20API-002f6c)](https://midtrans.com)

Discover the engineering behind KomeCore in the **[Developer Architecture Tutorial](docs/DEVELOPER_TUTORIAL.md)** and the **[Commerce System Design Matrix](docs/ARCHITECTURE.md)**

---

### Architecture Diagram

Below is a high-level overview of the system architecture:

![KomeCore Architecture Pattern](assets/architecture.jpg)

Patterns implemented in KomeCore:

* **Modular Monolith Architecture**
* **Hexagonal Architecture (Ports & Adapters)**
* **Pragmatic CQRS (Command-Query Separation)**
* **Compensating Transactions (Saga-Lite)**
* **Transactional Outbox Pattern**
* **Dual-Dialect Transactor Pattern**
* **Distributed Idempotency Key Pattern**
* **Atomic Conditional Updates (Zero-Lock Concurrency)**
* **Multi-Tier Cache-Aside Pattern**
* **Role-Based Access Control (RBAC)**

> See more in the **[Commerce System Design Matrix](docs/ARCHITECTURE.md)**.

### Features & System Capabilities

| Feature | Description |
| :--- | :--- |
| **Multi-Shop Catalog & Variants** | Manage multiple independent shops, rich product options and variants, media image galleries, and search-friendly URLs. |
| **Cart & Multi-Shop Checkout** | Customers can buy items from multiple shops in a single cart, with real-time calculations for shipping fees, discounts, and taxes. |
| **Discounts & Promotions Engine** | Create promotional vouchers and coupons with rules like minimum spend requirements, discount limits, and customer usage quotas. |
| **Payments & Automated Reconciliation** | Supports popular payment methods (bank transfers, QRIS, e-wallets) with automatic verification and instant order status updates. |
| **Fulfillment & Logistics** | Accurate courier rate calculations, waybill generation, and complete parcel tracking from packing to final delivery. |
| **Product Reviews & Customer Wishlists** | Verified buyer reviews and star ratings to build shopper trust, plus saved favorites and wishlist collections. |
| **Staff Administration & Analytics** | Management portal for merchant staff with role permissions, sales reports, and insights into top-performing products. |

> See more in the **[Developer Architecture Tutorial](docs/DEVELOPER_TUTORIAL.md)**.

### Dependencies & Tech Stack

| Category | Technology | Tool / Library |
| :--- | :--- | :--- |
| **Language & Runtime** | Go | Go 1.25+ |
| **HTTP Routing** | Chi | `github.com/go-chi/chi/v5` |
| **Relational Database** | PostgreSQL & SQL Server | `pgx/v5`, `go-mssqldb` |
| **Distributed Caching** | Redis | Redis 7+ (`go-redis/v9`) |
| **Payment Processing** | Midtrans | Core API |
| **Object Storage** | Cloud Storage | Supabase, GCS, S3 |
| **Auth & Security** | JWT & Cookies | `golang-jwt/v5`, `bcrypt` |
| **Observability** | Structured Logging | `uber-go/zap` |

---

### Quick Start

> **Environment Setup**: Copy `.env.example` to `.env`. Configure `DB_TARGET` (`postgres` · `supabase` · `sqlserver`), `STORAGE_PROVIDER` (`supabase` · `gcs` · `noop`), and `REDIS_ENABLED` (`true` · `false`). Schema migrations live in `migrations/<dialect>/` and seed data in `seeds/<dialect>/`.
>
> **Default Admin Account**: Running `make seed` or `make seed-dev` provisions a Super Admin with full platform permissions: `admin@komecore.com` / `AdminSecret123!`.

#### 1. Containerized Stack (Docker)

```bash
docker compose up --build -d
curl http://localhost:7129/health
# {"status":"healthy"}
```

#### 2. Local Development (Live Reload)

```bash
cp .env.example .env
make migrate && make seed-dev   # apply schema & development fixtures
make dev                        # run server with live reload (Air)
```

#### 3. Database & CLI Operations

```bash
make migrate            # apply pending migrations up
make migrate-version    # inspect current schema migration version
make seed               # populate base data (roles, super admin, couriers, payment guides)
make seed-dev           # populate base + development data (shops, products, customers)
make run                # run standalone server daemon
make test               # run full test suite
```

---

### API Ready-to-Test Collections

| Client / Format | Source File / Directory | Description & Instructions |
| :--- | :--- | :--- |
| **Postman** | [`api/postman_collection.json`](api/postman_collection.json) | Import directly into Postman Desktop or Web. Complete with collection variables and pre-configured request payloads. |
| **VS Code REST Client** | [`api/komecore.http`](api/komecore.http) | Open in VS Code and click **Send Request** directly in your editor. Zero external desktop client required. |
| **Bruno** | [`api/bruno/`](api/bruno) | Open the folder in the [Bruno API Client](https://www.usebruno.com/). Fully offline, Git-friendly `.bru` request files. |
| **OpenAPI / Swagger** | [`api/openapi.yaml`](api/openapi.yaml) & [`api/openapi-v1.0.0.yaml`](api/openapi-v1.0.0.yaml) | Import into Swagger UI, Redoc, or Insomnia. Full JSON schemas for all request/response DTOs and envelopes. |

For detailed steps on setting up environment variables, auth tokens, and testing order flows, see the [API README](api/README.md).

---

### Documentation

* **[Developer Architecture Tutorial](docs/DEVELOPER_TUTORIAL.md)**: Deep walkthrough explaining module architecture, the full request journey, and complex distributed lifecycles.
* **[Commerce System Design Matrix](docs/ARCHITECTURE.md)**: Comprehensive system design landscape for modern commerce platforms, detailing implemented patterns vs future evolution gaps.
* **[API Documentation & Collections](api/README.md)**: Complete guide to all endpoints with ready-to-use Postman collections, Bruno workspace, VS Code REST Client files, and frozen OpenAPI 3.0 specifications.

---

### Contributing

We encourage and support an active, healthy community of contributors, whether you are fixing a bug, writing documentation, or building new features! Details are in the **[Contributing Guidelines](CONTRIBUTING.md)** and **[Code of Conduct](CODE_OF_CONDUCT.md)**.

Looking for an easy task to get started? Explore issues tagged with [`good first issue`](https://github.com/deuterrr/komecore/issues?q=is%3Aissue+is%3Aopen+label%3A%22.contrib%3Agood+first+issue%22).

---

Released under the [MIT License](LICENSE).
