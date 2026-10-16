# API Documentation & Testing Collections

This directory contains pre-configured API collections, specifications, and environment files for testing and integrating with KomeCore.

All request bodies, query parameters, and headers are verified against the Go backend to ensure zero contract drift.

---

### Available Collections

All four collections provide complete coverage across all **95 endpoints** organized into 14 business domains:

| Format / Tool | Source File or Directory | Best Used For |
| :--- | :--- | :--- |
| **Postman** | [`postman_collection.json`](postman_collection.json) | Complete collection ready to import into Postman Desktop or Web. |
| **VS Code REST Client** | [`komecore.http`](komecore.http) | In-editor HTTP client with click-to-run requests via the REST Client extension. |
| **Bruno** | [`bruno/`](bruno/) | Offline, Git-friendly API client collection with local environment variables. |
| **OpenAPI / Swagger** | [`openapi.yaml`](openapi.yaml) & [`openapi-v1.0.0.yaml`](openapi-v1.0.0.yaml) | Official OpenAPI 3.0 specification for Swagger UI, Redoc, and code generators. |

---

### Testing Workflows

#### Postman
1. Open Postman, click **Import**, and select [`postman_collection.json`](postman_collection.json).
2. The collection imports as **Komecore API v1** with 14 organized domain folders (Authentication, Products, Carts, Orders, Payments, etc.).
3. Configure collection variables under the **Variables** tab:
   - `baseUrl`: `http://localhost:7129` (default local server)
   - `customerToken`: Paste customer JWT token (or leave empty if using cookies)
   - `staffToken`: Paste staff JWT token (or leave empty if using cookies)
   - Pre-populated test UUIDs: `shopId`, `productId`, `addressId`, `orderId`, `shipmentId`, etc.

#### VS Code REST Client
1. Install the **REST Client** extension (`humao.rest-client`) in VS Code.
2. Open [`komecore.http`](komecore.http).
3. Click **Send Request** directly above any request block to view headers and response bodies side-by-side.

#### Bruno
1. Open the **[Bruno API Client](https://www.usebruno.com/)**.
2. Click **Open Collection** and select the [`api/bruno`](bruno/) directory.
3. Select the **Local** environment from the top-right dropdown to load pre-configured variables.

#### Swagger & OpenAPI
- Import [`openapi.yaml`](openapi.yaml) into Swagger Editor, Swagger UI, or Redoc.
- All request payloads, response schemas, and error envelopes (`400`, `401`, `403`, `404`, `409`, `500`) are fully documented under `components.schemas`.

---

### Authentication Setup

KomeCore separates customer accounts from store staff accounts:

1. **Customer Accounts**: Created via `POST /api/v1/auth/signup` and verified via email OTP `POST /api/v1/auth/verify`.
2. **Staff Accounts**: Provisioned by administrators via `POST /api/v1/staff/{staffID}/accounts`.

When logging in through `/api/v1/auth/signin` or `/api/v1/auth/staff/signin`, the server automatically sets secure, HTTP-only session cookies. For API clients (Postman, Bruno, curl), you can also pass the token directly in the request header:
```http
Authorization: Bearer <your-jwt-token>
```

---

### Caching and Double-Submission Safety

#### Catalog Caching
Catalog endpoints use Redis to ensure lightning-fast product browsing:
- Product searches and listings are cached for 5 minutes.
- Individual product details are cached for 15 minutes.
- Updating or deleting products automatically clears the cache so shoppers always see fresh catalog data.

#### Order Double-Submission Protection
Mutating financial operations like placing an order (`POST /api/v1/orders`) require an `Idempotency-Key` header:
- **Missing Header**: Returns `400 Bad Request` explaining that the header is required.
- **In-Flight Lock**: If a network retry occurs while the first request is still processing, returns `409 Conflict` to prevent double charges.
- **Replay**: If a completed checkout request is re-sent with the same key within 24 hours, the saved response is replayed instantly with the header `X-Cache: HIT-IDEMPOTENCY` without charging the customer twice.

---

### Contract Verification

To confirm that all API collections, sample JSON payloads, and schemas match the Go backend code with zero discrepancies, run the contract test suite:

```bash
go test -v -count=1 ./test/contract/...
```

This automated suite verifies:
1. Full 95-endpoint parity across Postman, Bruno, `.http`, and OpenAPI.
2. OpenAPI 3.0 schema compliance for all request and response payloads.
3. Golden snapshot matching for critical business endpoints.
