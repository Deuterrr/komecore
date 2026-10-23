# 🌾KomeCore

Clean, scalable backend service engineered for modern e-commerce platforms in Go.

<img src="assets/placeholder.png" height=300>

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![Router](https://img.shields.io/badge/Router-Chi%20v5-blue)](https://github.com/go-chi/chi)
[![Database](https://img.shields.io/badge/Databases-PostgreSQL%20|%20SQL%20Server-4169E1?logo=postgresql)](https://www.postgresql.org)
[![Payments](https://img.shields.io/badge/Payments-Midtrans%20Core%20API-002f6c)](https://midtrans.com)

Discover the engineering behind KomeCore in the **[Developer Architecture Tutorial](docs/DEVELOPER_TUTORIAL.md)**

### License

This project is open-source software licensed under the [MIT License](LICENSE).

---

### Repository Architecture Layout

```text
komecore/
├── cmd/komecore/         # App entry point
├── internal/
│   ├── apperror/         # Error types and codes
│   ├── authctx/          # Authenticated actor context
│   ├── bootstrap/        # Dependency wiring and lifecycle
│   ├── config/           # Environment configuration
│   ├── httpx/            # HTTP utilities and middleware
│   ├── infra/            # External infrastructure adapters
│   └── modules/          # Business domains
│       └── <module>/
│              ├── <module>http/            # HTTP handlers and DTOs
│              └── <module>persistence/     # Database operations
├── api/                  # API specs and collections
├── docs/                 # Architecture and guides
├── migrations/           # Database migrations
├── pkg/                  # Reusable utilities
└── test/                 # Architecture and integration tests
```

### Features & System Capabilities

#### E-Commerce Domain Features

| Feature | Description |
| :--- | :--- |
| **Multi-Shop Catalog & Variants** | Manage multiple independent shops, rich product options and variants, media image galleries, and search-friendly URLs. |
| **Cart & Multi-Shop Checkout** | Customers can buy items from multiple shops in a single cart, with real-time calculations for shipping fees, discounts, and taxes. |
| **Discounts & Promotions Engine** | Create promotional vouchers and coupons with rules like minimum spend requirements, discount limits, and customer usage quotas. |
| **Payments & Automated Reconciliation** | Supports popular payment methods (bank transfers, QRIS, e-wallets) with automatic verification and instant order status updates. |
| **Fulfillment & Logistics** | Accurate courier rate calculations, waybill generation, and complete parcel tracking from packing to final delivery. |
| **Product Reviews & Customer Wishlists** | Verified buyer reviews and star ratings to build shopper trust, plus saved favorites and wishlist collections. |
| **Staff Administration & Analytics** | Management portal for merchant staff with role permissions, sales reports, and insights into top-performing products. |

#### System & Architectural Features

| Capability | Description |
| :--- | :--- |
| **Dual-Dialect Database Support** | Works seamlessly on both PostgreSQL and Microsoft SQL Server without changing any business logic. |
| **Flash-Sale Inventory Protection** | Prevents overselling during high-traffic sales by holding stock during checkout and automatically restoring it if payment fails. |
| **Double-Charge Protection (Idempotency)** | Shields customers from duplicate orders and accidental double-charges caused by double-clicks or dropped internet connections. |
| **Guaranteed Event Delivery (Outbox)** | Ensures background tasks (like order confirmation emails and notifications) are never lost, even if a service temporarily restarts. |
| **Smart Catalog Search** | Fast product discovery that understands word variations and ranks the most relevant products first. |
| **Customer & Staff Security** | Safe authentication for shoppers and store admins, featuring Google one-click login, email verification codes, and secure sessions. |
| **Built-In Reliability & Safety** | Protects the API from traffic abuse, hides sensitive internal errors from clients, and includes automated contract testing. |

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

> **Environment Setup**: Configure `DB_TARGET` (`postgres` · `supabase` · `sqlserver`), `STORAGE_PROVIDER` (`supabase` · `gcs` · `noop`), and `REDIS_ENABLED` (`true` · `false`) directly in `.env`. Database migrations live in `migrations/<dialect>/`.

#### 1. Docker Compose

```bash
docker compose up --build -d
curl http://localhost:7129/health
# {"status":"healthy"}
```

#### 2. Live Reload / Hot Reload (`air`)

```bash
cp .env.example .env
make migrate && make seed
make dev   # or run `air`
```

#### 3. Makefile Automation

```bash
cp .env.example .env
make migrate        # or: make migrate-supabase / make migrate-sqlserver
make seed
make run
make test
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

### Community & Contributing

We warmly welcome contributions from the community! Whether you are fixing a bug, writing documentation, or building new features, please review our contribution and behavioral standards:

* **[Contributing Guidelines](CONTRIBUTING.md)**: Complete local setup instructions, Docker workflows, testing conventions, and pull request requirements.
* **[Code of Conduct](CODE_OF_CONDUCT.md)**: Community standards, behavioral pledges, and reporting channels to ensure a welcoming and inclusive environment.
* **Good First Issues**: Looking for an easy task to get started? Explore issues tagged with [`good first issue`](https://github.com/deuterrr/komecore/issues?q=is%3Aissue+is%3Aopen+label%3A%22.contrib%3Agood+first+issue%22).

### More Documentation

* **[Developer Architecture Tutorial](docs/DEVELOPER_TUTORIAL.md)**: Deep walkthrough explaining module architecture, the full request journey, and complex distributed lifecycles.
* **[API Documentation & Collections](api/README.md)**: Complete guide to all endpoints with ready-to-use Postman collections, Bruno workspace, VS Code REST Client files, and frozen OpenAPI 3.0 specifications.
