# Komecore API Documentation & Testing Collections

This folder contains API specifications, request collections, and environment files for testing and integrating with the **Komecore** backend.

All collections and specifications are **100% verified against Go delivery DTO structs and Chi v5 router definitions**.

---

## 📂 Directory Contents

| File / Folder | Purpose | Best Used With |
| :--- | :--- | :--- |
| **[`openapi.yaml`](openapi.yaml)** | Complete OpenAPI 3.0.3 YAML Specification with full `components.schemas` | **Swagger UI / Redoc / Insomnia** |
| **[`postman_collection.json`](postman_collection.json)** | Complete Postman v2.1 API Collection (58 endpoints across 12 domains) | **Postman Desktop / Web** |
| **[`komecore.http`](komecore.http)** | Plain-text HTTP Client request file matching Go DTOs | **VS Code (REST Client Extension)** |
| **[`bruno/`](bruno/)** | Bruno API Client Collection & Environment (58 `.bru` requests) | **Bruno Desktop App** |

---

## 🚀 Step-by-Step Testing Guide

### 1. Testing with Postman
1. Open **Postman**.
2. Click **Import** (top left) → Select file → Choose [`api/postman_collection.json`](postman_collection.json).
3. Postman will import **`Komecore API v1`** organized into 12 domain folders:
   - `01. Health & Infrastructure`
   - `02. Authentication & Identity`
   - `03. User & Profile`
   - `04. Saved Addresses`
   - `05. Staff Administration`
   - `06. Catalog & Products`
   - `07. Shops & Inventories`
   - `08. Cart & Checkout`
   - `09. Couriers & Logistics`
   - `10. Orders & Tracking`
   - `11. Shipments`
   - `12. Payments & Webhooks`
4. Set up collection variables (Click `Komecore API v1` → **Variables** tab):
   - `baseUrl`: `http://localhost:7129` (Default)
   - `customerToken`: Paste customer JWT token (or rely on HTTP-only cookies)
   - `staffToken`: Paste staff JWT token (or rely on HTTP-only cookies)
   - Pre-populated test UUIDs: `shopId`, `productId`, `addressId`, `orderId`, `shipmentId`, `methodId`.

---

### 2. Testing with VS Code REST Client
1. Install the **REST Client** extension (`humao.rest-client`) in VS Code.
2. Open [`api/komecore.http`](komecore.http).
3. Click **`Send Request`** right above any request line.
4. Response bodies and headers render side-by-side in your editor.

---

### 3. Testing with Bruno
1. Open **[Bruno](https://www.usebruno.com/)**.
2. Click **Open Collection** → Select the [`api/bruno`](bruno/) directory.
3. In the top right environment dropdown, select **Local**.
4. All 58 endpoints are available in the sidebar with matching JSON payloads and headers.

---

### 4. Viewing OpenAPI / Swagger Specs
- Import [`api/openapi.yaml`](openapi.yaml) into [Swagger Editor](https://editor.swagger.io/) or any local Swagger/Redoc container.
- All request bodies, query parameters, multipart form schemas, and error responses (`400`, `401`, `403`, `404`, `409`) are defined under `components.schemas`.

---

## 🔐 Auth & Identity Mechanics

Komecore uses a dual-layer authentication model:

1. **Dual Actor System**:
   - **Customer**: Registered via `POST /api/v1/auth/signup` and OTP verification `POST /api/v1/auth/verify`.
   - **Staff / Admin**: Managed by administrators via `POST /api/v1/staff/{staffID}/accounts`.
2. **Session Delivery**:
   - On successful sign-in (`/api/v1/auth/signin` or `/api/v1/auth/staff/signin`), the server sets secure HTTP-only cookies:
     - `kome_customer_access` / `kome_customer_refresh` (Customer)
     - `kome_staff_access` / `kome_staff_refresh` (Staff)
   - The authenticator middleware also accepts `Authorization: Bearer <token>` in request headers for API clients like Postman and Bruno.

---

## ⚡ Distributed Caching & Idempotency Mechanics (Phase 2)

### 1. Redis Catalog Cache-Aside Pattern
Catalog read endpoints employ high-performance Redis 7 caching to minimize PostgreSQL query latency:
- **`GET /api/v1/products`**: Caches filtered and paginated search queries (TTL: 5 minutes, key: `cache:products:list:<query_hash>`).
- **`GET /api/v1/products/{slug}`**: Caches individual product detail representations (TTL: 15 minutes, key: `cache:product:slug:<slug>`).
- **Proactive Invalidation**: Mutating operations (`POST /api/v1/products`, `PUT /api/v1/products/...`, `DELETE /api/v1/products/...`) automatically trigger pattern-based eviction (`cache:product*`) across all active nodes.

### 2. Request Idempotency & Concurrency Guards
Mutating operations vulnerable to double-submission or transient network retries enforce distributed idempotency:
- **`POST /api/v1/order`**: Requires the `Idempotency-Key` header (UUID or string up to 128 characters).
- **Behavior & Status Codes**:
  - **Missing / Invalid Header**: Returns `400 Bad Request` (`"Idempotency-Key header is required"`).
  - **In-Flight Lock**: If a concurrent request with the same key is currently running, returns `409 Conflict` (`"a request with this idempotency key is currently processing"`).
  - **Replay Execution**: Once completed, subsequent requests with the same key within 24 hours replay the saved response with the `X-Cache: HIT-IDEMPOTENCY` header without re-executing database operations.

---

## 🧪 Automated Contract Verification

To ensure API documentation and sample payloads never drift from the Go backend DTOs, a contract test is included in the test suite:

```bash
go test -v -run TestAPIContract ./internal/bootstrap
```

This tests:
1. Complete coverage of Postman collection endpoints and valid JSON bodies.
2. OpenAPI 3.0.3 specification completeness and schema resolution.
3. Type compatibility between JSON sample payloads and the actual Go DTO structs.
