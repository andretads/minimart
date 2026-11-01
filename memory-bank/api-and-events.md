# API and Events

## Response Contract

- Every JSON response uses the shared envelope (`backend/pkg/response`):

  ```json
  {
    "time": "2026-01-01T00:00:00Z",
    "success": true,
    "data": {},
    "error": null,
    "meta": {}
  }
  ```

- Errors use `error: {code, message}`; `code` mirrors the HTTP status. `error.fields` is an optional map of field-level validation messages. Success is `2xx`; pagination and filters appear under `meta` where applicable. `data`, `error`, and `meta` are omitted when empty; the `meta` fields (`page`, `page_size`, `total`, `total_pages`) are omitted when zero.
- Customer-facing `error.message` text is in Brazilian Portuguese (pt-BR). Sentinel definitions carry a domain namespace prefix (e.g. `cart: carrinho não encontrado`); internal-only diagnostics stay in English. The Angular client matches Portuguese substrings of these messages for friendly copy.
- Layer mapping: HTTP statuses are decided only in presentation (`respondDomainError`): validation 400, unauthenticated 401, forbidden 403, not found 404, stale version / duplicate / idempotency conflict / insufficient stock 409, rate limit 429, unexpected 500.
- Local endpoints `/livez`, `/readyz`, and `/metrics` are not enveloped; `readyz` returns `{time, status, checks}` where `status` is `healthy`, `degraded` (a non-critical dependency is down → HTTP 200), or `unhealthy` (a critical dependency is down → HTTP 503), and `checks` maps a dependency name to `{status, latency?, error?}`. The **worker** exposes its own `/livez`, `/readyz`, `/metrics` on `WORKER_HTTP_ADDR` (default `:8081`); the **gateway** exposes local ones on its own address.

## Authentication and Authorization

- **Auth is cookie-based.** Login returns user data plus the access-token expiry and sets two encrypted cookies:
  - `mm_at` — access token, 15m TTL.
  - `mm_rt` — refresh token, 7d TTL (30d when `remember_me` is true and `AUTH_REMEMBER_ME_REFRESH_TOKEN_TTL` > 0).
- `POST /v1/auth/refresh` performs **sliding renewal**: it issues a new access token and a new refresh token (preserving the remember-me choice). Refresh tokens are not revoked, so older still-unexpired refresh tokens remain valid.
- Login and refresh return `access_token_expires_at` (RFC 3339) and `expires_in` (seconds). The SPA should prefer `expires_in` to avoid client clock skew when scheduling proactive refresh.
- Cookies are AES-GCM encrypted (`COOKIE_ENCRYPTION_KEY`, base64 32-byte key), `HttpOnly`, `SameSite=Lax`, `Path=/`, and `Secure` in production. Optional `COOKIE_DOMAIN` supports a shared parent domain.
- The browser must send cookies: same-origin, or credentialed CORS with an explicit origin. The SPA never stores tokens itself.
- The access cookie is validated by `backend/pkg/auth.Middleware`. Missing/invalid/expired access tokens produce 401 with `WWW-Authenticate: Bearer error="invalid_token"`.
- `authz.Middleware` resolves the registered route pattern and requires the permission in `backend/pkg/authz/policy.go`. Public routes need no token/permission; optional-auth routes pass without a token and expose claims when present; routes absent from the policy are **denied (fail-closed)**.
- **Current auth-path caveats**:
  - `/v1/auth/logout` is public in both authz and the server's public-auth list. It clears cookies even when the access token is missing or expired.
  - `/v1/auth/me` is optional at the middleware; its handler returns 401 when no claims are present.
  - `/v1/categories` requires `catalog:read` (GET) / `catalog:write` (POST). `/v1/categories/{id}` is protected. Public browsing uses `/v1/showcase/categories`.
  - Catalog review routes: `POST /v1/products/{id}/reviews` maps to `reviews:write`, held by the `customer` and `admin` roles; `GET /v1/products/{id}/reviews` maps to `catalog:read`, held by `customer`, `support` and `admin`. The public showcase reviews endpoint remains separate and needs no auth.
  - `/v1/shipping/*` is not public. `GET /v1/shipping/methods` therefore requires authentication and `shipping:read`, unlike the public showcase/category reads.
  - **Public path lists** (must stay in sync): `backend/pkg/auth/paths.go:PublicRoutePaths` holds 10 entries (`/livez`, `/readyz`, `/metrics`, `/v1/auth/signup|login|refresh|logout`, `/v1/showcase/products`, `/v1/showcase/products/`, `/v1/showcase/categories`); `backend/pkg/authz/policy.go:PublicPaths` holds 12 (the same plus `/swagger/`, `/v1/showcase/products/{id}`, and `/v1/showcase/products/{id}/reviews`); `authz.OptionalAuthPaths` holds one (`/v1/auth/me`). The cross-module test `backend/internal/routes/routes_test.go` asserts that every `authz.PublicPaths` entry is also public for the authentication layer.
  - **RBAC roles**: `customer`, `support`, `admin`. Permissions seeded: catalog, customer, order, payment, notification, account, address, avatar, cart, wishlist, and shipping reads/writes per the initial migration (`20261106231155_initial_schema.sql`) (migrations 15–17, 19, 26 consolidated). `catalog:write` and shipping admin writes are admin-only; `reviews:write` is held by `customer` and `admin`; customers hold both `customers:read` and `customers:write`; `cart:*`/`wishlist:*` are held by all three roles.
- **Rate limiting**: global token bucket keyed by `Idempotency-Key`, falling back to client IP; 429 + `Retry-After` + `X-RateLimit-Remaining`. Installed in both gateway (if Redis is reachable) and server. The Redis path is wrapped in a circuit breaker and a short per-call timeout, and fails open.
- **Idempotency-Key** is required for `POST /v1/orders`, `POST /v1/payments`, `POST /v1/cart/checkout`, and `POST /v1/shipping/quotes`.
- **CSRF**: no separate token; protection is `SameSite=Lax` plus CORS policy.
- **Request correlation**: the server and gateway honor an inbound `X-Request-ID` header (or generate one), echo it on the response, and stamp it on the access-log line.

## HTTP Endpoints

### System and docs

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/livez` | public | process alive (server, worker, gateway) |
| GET | `/readyz` | public | dependency checks, 2s timeout (server and worker both check Postgres read/write pools, Mongo, and Redis; Redis is non-critical by default and reports `degraded` with HTTP 200 when unavailable; Kafka and S3 are not probed; gateway is dependency-free). Body: `{time, status, checks}` |
| GET | `/metrics` | public | Prometheus (server, worker, gateway) |
| GET | `/swagger/*` | public (non-production only) | generated docs |

The **gateway** exposes `/livez`, `/readyz`, and `/metrics` locally and proxies everything else unchanged. It does not authenticate or rewrite payloads.

### Auth (`/v1/auth`)

| Method | Path | Access | Request / response |
| --- | --- | --- | --- |
| POST | `/signup` | public | `{username, email, password}` → 201 user; creates customer and `customer` role; does not log in |
| POST | `/login` | public | `{username, password, remember_me?}` → user data + `access_token_expires_at`, `expires_in`, and `mm_at`/`mm_rt` cookies |
| POST | `/refresh` | public | reads `mm_rt`, renews both cookies, returns `{status, access_token_expires_at, expires_in}` |
| POST | `/logout` | public | clears both cookies, even without a valid access token |
| POST | `/change-password` | `accounts:write`, self | `{current_password, new_password, confirm_password}` |
| GET | `/me` | optional auth, 401 without claims | `{id, username, email, created_at, access_token_expires_at, expires_in}` |

Constraints: username required ≤200; password 6–72 bytes; email required, valid, ≤254, unique case-insensitively when present.

### Catalog (`/v1/products`)

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| POST | `/v1/products` | `catalog:write` | JSON or multipart; `name`, `price_cents`, `stock`, `category_id`, `sku`, `description`, `details` (JSON array). Images are added separately. |
| GET | `/v1/products` | `catalog:read` | optional `category_id` |
| GET | `/v1/products/{id}` | `catalog:read` | returns `ETag: "<version>"`; supports `If-None-Match` → 304 |
| PATCH | `/v1/products/{id}` | `catalog:write` | JSON or multipart; `version` required (or `If-Match` wins); updates name/price/category/SKU/description/details, not stock. Images are managed separately. |
| POST | `/v1/products/{id}/images` | `catalog:write` | multipart `image`; appends image, assigns next position, bumps product version |
| GET | `/v1/products/{id}/images` | `catalog:read` | images ordered by position; lowest position is primary |
| DELETE | `/v1/products/{id}/images/{image_id}` | `catalog:write` | removes image and bumps product version |
| POST | `/v1/products/{id}/reviews` | `reviews:write` (`customer` and `admin` roles) | `{description, rating 1–5}`; the customer is taken from the session, not the body; created `pending` and moderated asynchronously |
| GET | `/v1/products/{id}/reviews` | `catalog:read` | approved reviews only, newest first; paginated (`page` default 1, `page_size` default 20/max 100); pagination in `meta` |

Product response: `id, name, price_cents, stock, version, created_at, updated_at, category_id, sku?, description?, details?, total_rating?, total_reviews?, images[]`. Each image carries `id, product_id, url, content_type, size_bytes, position, created_at, updated_at`; `url` is a presigned GET URL. A product category must exist (`category.GetCategory`); duplicate SKU and invalid rating/empty description are 400; a stale version and a duplicate review are 409.

### Categories (`/v1/categories`)

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| POST | `/v1/categories` | `catalog:write` | `{name, icon}` |
| GET | `/v1/categories` | `catalog:read` | list |
| GET | `/v1/categories/{id}` | `catalog:read` | detail, ETag |
| PATCH | `/v1/categories/{id}` | `catalog:write` | `{name, icon, version}` or `If-Match` |
| DELETE | `/v1/categories/{id}` | `catalog:write` | `{version}` required; blocked if products reference it |

Category fields: `id, name, icon, created_at, updated_at, version`; name is unique case-insensitively (≤100), icon ≤500.

### Customer (`/v1/customers`)

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| POST | `/v1/customers` | `customers:write` | `{user_id, display_name}`; 409 on duplicate user id |
| GET | `/v1/customers/{id}` | `customers:read` | ETag; supports `If-None-Match` |
| GET | `/v1/customers?user_id=` | `customers:read` | ETag |
| PATCH | `/v1/customers/{id}` | `customers:write` | `{display_name, version}`; `If-Match` wins |
| POST | `/v1/customers/{id}/deactivate` | `customers:write` | idempotent, bumps version |

The default `customer` role holds `customers:read` (migration 15), so the profile's own customer fetch through these read routes succeeds; support and admin retain the same read access.

### Addresses (`/v1/addresses`)

All require authentication and the listed permission. The handler derives the caller's user from the cookie and resolves the customer through `customer.EnsureCustomer`; ownership is enforced in the application layer (mismatch → 404).

| Method | Path | Permission |
| --- | --- | --- |
| POST | `/v1/addresses` | `addresses:write` |
| GET | `/v1/addresses` | `addresses:read` |
| GET | `/v1/addresses/{id}` | `addresses:read` |
| PATCH | `/v1/addresses/{id}` | `addresses:write` (requires `version`) |
| DELETE | `/v1/addresses/{id}` | `addresses:write` (requires `version`) |
| POST | `/v1/addresses/{id}/default` | `addresses:write` |

Address fields: `id, customer_id, label, full_name, street, city, state, zip, country, phone, is_default, version, created_at, updated_at`. Exactly one default per customer (`addresses_customer_default_unique`); a second create with `is_default` is 409; setting a new default clears the old one. `country` is normalized on write: localized names such as "Brasil"/"Brazil" become `BR` and any two-letter value is upper-cased, but other values are stored unchanged, so the field is **not** guaranteed to be an ISO-3166 alpha-2 code. `phone` is capped at 30 characters.

### Avatar (`/v1/avatars`)

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/v1/avatars/{account_id}` | `avatars:read` | metadata + presigned `image_url`; not self-only |
| PUT | `/v1/avatars/{account_id}` | `avatars:write` + path must equal claim | multipart `image`; JPEG/PNG/WebP/GIF, size bounded; upsert |

### Cart (`/v1/cart`)

Authenticated (`cart:read` / `cart:write`). Identity is the authenticated user from the access-token cookie; the cart is scoped to that user.

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/v1/cart` | `cart:read` | creates an active cart when none exists |
| POST | `/v1/cart/items` | `cart:write` | `{product_id, quantity}`; existing product increments quantity |
| PATCH | `/v1/cart/items/{product_id}` | `cart:write` | `{quantity}` |
| DELETE | `/v1/cart/items/{product_id}` | `cart:write` | remove item |
| DELETE | `/v1/cart` | `cart:write` | clear, keeps cart active, 204 |
| POST | `/v1/cart/checkout` | `cart:write` | **`Idempotency-Key` required**; body `{shipping_quote_id, address_id}`; returns the placed order |

### Orders (`/v1/orders`)

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| POST | `/v1/orders` | `orders:write` | **`Idempotency-Key` required**; `{user_id, items:[{product_id, quantity}]}`; reserves stock before save |
| GET | `/v1/orders?user_id=&page=&page_size=` | `orders:read` | paginated user order list, newest first; `page` 1-based, `page_size` default 20/max 100 |
| GET | `/v1/orders/{id}` | `orders:read` | includes status/items and the shipping snapshot |
| POST | `/v1/orders/{id}/cancel` | `orders:write` | allowed only while unpaid; idempotency conflicts 409 |

Order statuses: `placed`, `paid`, `payment_failed`, `payment_unknown`, `cancelled`. `paid`, `payment_failed`, `cancelled` are absorbing. Cancelling a paid order is 409; recancelling is 409; unknown product/insufficient stock are 4xx (stock conflict 409). The order response carries the shipping snapshot stored with the order: `shipping_quote_id`, `shipping_method_id`, `shipping_method_name`, `shipping_cost_cents`, and `destination_snapshot` (`label`, `full_name`, `street`, `city`, `state`, `zip`, `country`, `phone`). The snapshot is written by the cart checkout path; an order placed directly through `POST /v1/orders` has no shipping data. A placed order also carries `auto_cancel_at` (RFC3339), set 5 minutes after creation.

### Shipping (`/v1/shipping`)

All shipping routes are authenticated. Admin CRUD and fulfillment transitions require shipping permissions. Quote/shipment **creation** and the tracking-history read enforce ownership; the quote/shipment detail reads and the shipment list are not owner-scoped (see gaps).

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/v1/shipping/methods` | `shipping:read` | list methods |
| POST | `/v1/shipping/methods` | `shipping:write` | `{name, description, active, estimated_min_days, estimated_max_days}` |
| PATCH | `/v1/shipping/methods/{id}` | `shipping:write` | update method |
| POST | `/v1/shipping/zones` | `shipping:write` | `{name, country, state, city, postal_code_start, postal_code_end}` |
| PATCH | `/v1/shipping/zones/{id}` | `shipping:write` | update zone |
| POST | `/v1/shipping/rates` | `shipping:write` | `{zone_id, method_id, base_cost_cents, free_threshold_cents, active}` |
| PATCH | `/v1/shipping/rates/{id}` | `shipping:write` | update rate |
| POST | `/v1/shipping/quotes` | `shipping:read` | `{method_id, address_id, subtotal_cents?}` + required `Idempotency-Key` header; customer resolved from the cookie; returns quote snapshot (cost, subtotal and free threshold); same key + same customer/method/address/subtotal replays the original quote (201), changed parameters → 409 |
| GET | `/v1/shipping/quotes/{id}` | `shipping:read` | quote detail (not owner-scoped) |
| GET | `/v1/shipping/shipments` | `shipping:read` | list shipments for the authenticated user; optional `order_id` returns a single-item list; a `user_id` query parameter overrides the caller (not owner-scoped) |
| GET | `/v1/shipping/shipments/{id}` | `shipping:read` | shipment detail (not owner-scoped) |
| GET | `/v1/shipping/shipments/{id}/events` | `shipping:read` | tracking history of a shipment, oldest first (owner-scoped, 403 on mismatch) |
| POST | `/v1/shipping/shipments/{id}/process` | `shipping:write` | `pending` → `processing` |
| POST | `/v1/shipping/shipments/{id}/dispatch` | `shipping:write` | `processing` → `shipped`; `{tracking_code, carrier}` |
| POST | `/v1/shipping/shipments/{id}/deliver` | `shipping:write` | `shipped` → `delivered` |
| POST | `/v1/shipping/shipments/{id}/events` | `shipping:write` | `{shipment_id, status, description, location, occurred_at}`; append-only |

Quote fields: `id, customer_id, method_id, destination_snapshot, shipping_cost_cents, subtotal_cents, free_threshold_cents, estimated_min_days, estimated_max_days, status, expires_at, created_at` (the persisted `address_id`/`idempotency_key` are not returned). Shipment fields: `id, order_id, user_id, method_id, tracking_code, carrier, shipping_cost_cents, destination_snapshot, status, shipped_at?, delivered_at?, created_at, updated_at`. Shipment statuses: `pending`, `processing`, `shipped`, `delivered`, `failed`.

### Payments (`/v1/payments`)

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| POST | `/v1/payments` | `payments:write` | **`Idempotency-Key` required**; `{order_id, user_id, amount_cents, currency, method}` |
| GET | `/v1/payments/{id}` | `payments:read` | read-only |
| POST | `/v1/payments/{id}/refund` | `payments:write` | refunds a `succeeded` payment; requires its order to be `cancelled` |

States: `pending`, `succeeded`, `failed`, `unknown`, `refunded`. Provider decline → `failed`; timeout/technical error → `unknown`. Retrying the same key/params replays; different params → 409. Payment creation loads the order through the order contract and verifies `user_id` (owner), `amount_cents` (against the order total), and a non-empty order `currency` before charging; a missing order → 404, owner/amount/currency mismatches → 400. The order domain carries no currency today, so the currency comparison normally does not fire. A fake provider can force decline/timeout via `PAYMENT_FAKE_DECLINE_AMOUNT_CENTS` / `PAYMENT_FAKE_TIMEOUT_AMOUNT_CENTS`.

Refunds reverse a captured charge on a cancelled order to `refunded` and publish `payment-refunded`. Three paths converge on the same outcome: the `order-cancelled` consumer (`payment-refund` group) refunds every `succeeded` payment of the order (idempotent, tolerant of lost races); CreatePayment re-checks the order after capture and refunds inline when it was cancelled mid-charge (keeping the charge reference on the still-pending payment when the refund fails, so the reconciler retries); and the reconciler refunds a captured charge it confirms on a cancelled order instead of finalizing it. `POST /v1/payments/{id}/refund` is the manual trigger — it requires the order to be `cancelled` (409 otherwise), replays an already-refunded payment, rejects non-`succeeded` payments with 409, and maps a provider refund failure to 502.

### Notifications (`/v1/notifications`)

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/v1/notifications` | `notifications:read` | newest first; scoped to the authenticated user |
| GET | `/v1/notifications/{id}` | `notifications:read` | scoped to the authenticated user (404 otherwise) |
| POST | `/v1/notifications/{id}/read` | `notifications:write` | mark a notification as read |
| POST | `/v1/notifications/{id}/unread` | `notifications:write` | mark a notification as unread |
| POST | `/v1/notifications/read-all` | `notifications:write` | mark all of the user's notifications as read |
| DELETE | `/v1/notifications/{id}` | `notifications:write` | delete a notification |
| GET | `/v1/notifications/preferences` | `notifications:read` | authenticated user only; defaults when unsaved |
| PUT | `/v1/notifications/preferences` | `notifications:write` | full replacement; all four booleans required |

Preference fields/defaults: `email_enabled=true`, `sms_enabled=false`, `push_enabled=false`, `marketing_enabled=false`. The preferred channel set is gated by category: transactional notifications need the channel toggle, marketing notifications additionally need `marketing_enabled`, and `critical` bypasses preferences (no current type maps to it). Preferences live in Mongo keyed by user id; a missing document means defaults. Legacy `order_updates`/`promotions`/`security_alerts`/`newsletter` documents are mapped on read.

### Wishlist (`/v1/wishlist`)

Authenticated (`wishlist:read` / `wishlist:write`). Identity is the authenticated user from the access-token cookie; the wishlist is scoped to that user.

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/v1/wishlist` | `wishlist:read` | creates wishlist if none exists |
| POST | `/v1/wishlist/items` | `wishlist:write` | `{product_id}` to add item |
| DELETE | `/v1/wishlist/items/{product_id}` | `wishlist:write` | remove item |
| DELETE | `/v1/wishlist` | `wishlist:write` | clear wishlist |

### Showcase (`/v1/showcase/products`)

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/v1/showcase/products` | public | keyset pagination: `page_size` (default 20, max 100), optional `cursor` (opaque token from `meta.next_cursor`) and `category_id`; `meta` carries `page_size`, `next_cursor`, `has_more`. Items carry `stock`, `total_rating`, `total_reviews`, and `image_url` (primary image, presigned). Ordered by `created_at DESC, id DESC`. |
| GET | `/v1/showcase/products/{id}` | public | `{id, name, price_cents, category_id, stock, sku?, description?, details?, total_rating, total_reviews, image_url?, images[]}`; each image carries a presigned `url`. |
| GET | `/v1/showcase/products/{id}/reviews` | public | approved reviews only, newest first; paginated (`page` default 1, `page_size` default 20/max 100); items are `{id, product_id, customer_id, description, rating, status, created_at}` and pagination is in `meta` |
| GET | `/v1/showcase/categories` | public | list of categories `{id, name, icon}` |

Showcase image URLs are presigned with `S3_PRESIGN_TTL` and can expire; clients must fall back to a placeholder. `total_rating`/`total_reviews` are denormalized counters updated only when moderation approves a review; with the moderation provider down, reviews stay `pending` and the public reviews list stays empty. Showcase reads only the replica. The catalog-owned review page read is cached in Redis under `catalog:reviews:<product_id>:<limit>:<offset>` (1m TTL); showcase has no cache of its own.

## Cross-Domain Contracts

Contracts are the **only** in-process cross-domain interface. They live in `backend/pkg/module/contracts`; adapters are `presentation/module/*.go`; clients define their own sentinels (never re-export domain errors). `Verify()` fails boot if any of the seven slots is nil.

| Slot | Published by | Methods / DTO | Sentinels |
| --- | --- | --- | --- |
| `Contracts.CategoryClient` | `category` | `GetCategory(ctx, id) (Category{ID,Name,Icon}, error)` | `ErrCategoryNotFound` |
| `Contracts.CatalogClient` | `catalog` | `GetProduct(ctx,id) (Product,error)`; `ReserveStock(ctx, orderID, []LineItem)`; `ReleaseStock(ctx, orderID)` | `ErrProductNotFound`, `ErrInsufficientStock` |
| `Contracts.CustomerClient` | `customer` | `EnsureCustomer(ctx,userID,displayName?)`; `GetCustomerByID`; `GetCustomerByUserID` | `ErrCustomerNotFound` |
| `Contracts.OrderClient` | `order` | `PlaceOrder(ctx, PlaceOrderRequest) (Order, error)`; `GetOrder(ctx,id)` | `ErrOrderNotFound`, `ErrIdempotencyConflict`, `ErrOrderInProgress`, `ErrInsufficientStock`, `ErrProductNotFound`, `ErrNoItems`, `ErrInvalidQuantity`, `ErrIdempotencyKeyRequired` |
| `Contracts.ShippingClient` | `shipping` | methods/zones/rates CRUD reads; `GetQuote`/`CreateQuote`/`ValidateQuote`; `GetShipment`/`GetShipmentByOrderID`/`ListShipments`; `GetTrackingEvents` | quote/method/zone/rate/shipment sentinels (see Shipping) |
| `Contracts.AddressClient` | `address` | `GetAddress(ctx, GetAddressRequest{ID, CustomerID}) (Address, error)` | `ErrAddressNotFound` |
| `Contracts.AuthClient` | `auth` | `GetAccountEmail(ctx, userID) (string, error)` | `ErrAccountNotFound`, `ErrEmailNotFound` |

`contracts.Modules` embeds six of these (catalog, category, customer, order, shipping, address); `contracts.Contracts` adds `AuthClient`. Consumers: address→customer; auth→customer (signup provisioning); cart→catalog+customer+order+shipping; catalog→category+customer (reviews); notification→auth+customer+address (recipient email and phone); order→catalog+customer; payment→order (pre-charge order validation and refund checks); shipping→address+customer; wishlist→catalog+customer. `avatar`, `cart`, `notification`, `payment`, `showcase`, and `wishlist` publish nothing. Contracts carry dependency-free DTOs; no domain types, no imports of another module's application/infrastructure.

## Kafka Topics and Events

Transport is a thin wrapper over Sarama/Watermill (`backend/pkg/kafka`); domains depend on `Publisher` and `func(ctx, []byte) error`, never the libraries. Delivery is **at-least-once**.

A failed handler is retried in-process with exponential backoff and full jitter (`KAFKA_CONSUMER_MAX_ATTEMPTS`, default 5; base/max delay default 500ms/30s). A permanent error (explicit `kafka.Permanent`, malformed JSON) skips retries. When retries are exhausted, or immediately for a permanent error, the message is republished to `<topic>.dlq` with unchanged payload and key, then the original is acknowledged. The DLQ publish is synchronous: if it fails the original is left unacknowledged so Kafka redelivers it. Shutdown (`context.Canceled`) never dead-letters. If a consumer loop stops unexpectedly the worker restarts it with capped backoff (`KAFKA_CONSUMER_RESTART_MIN_BACKOFF`/`MAX_BACKOFF`).

Backpressure: each (topic, group) runs `KAFKA_CONSUMER_CONCURRENCY` supervised replicas (default 1). Watermill's unbuffered subscriber channel provides natural backpressure; an opt-in pause gate (`KAFKA_CONSUMER_PAUSE_ENABLED`) pauses reads when the write/read pool is saturated (`KAFKA_CONSUMER_PAUSE_POOL_THRESHOLD`, default 0.9), so Kafka accumulates the excess. Consumer lag is sampled by the lag monitor (`KAFKA_CONSUMER_LAG_ENABLED`, default true; `KAFKA_CONSUMER_LAG_INTERVAL`, default 30s) and exported as `minimart_kafka_consumer_lag`.

DLQ metadata travels as `x-minimart-*` Kafka headers: `original-topic`, `original-partition`, `original-offset`, `consumer-group`, `attempts`, `disposition` (`permanent`/`exhausted`), `first-failed-at`, `failed-at`, `error` (truncated), `dlq-version`, `replay-count`. Replay with `make -C backend dlq-replay topic=<original-topic>` (`dry_run=1` to inspect); it republishes the unchanged payload to the original topic, strips old triage headers, adds `replayed-from`/`replayed-at` and increments `replay-count`, and commits progress under the `dlq-replay` group. A message is parked once it reaches `--max-replays` (default 3). Compose provisions each source topic with 6 partitions and each `<topic>.dlq` with 14-day retention.

| Topic | Payload (`backend/pkg/message`) | Producer | Consumers (group) |
| --- | --- | --- | --- |
| `order-placed` | `OrderPlaced{order_id,user_id,customer_id?,total_cents,item_count}` | order outbox | order (`order-confirmation`), notification (`notification`) |
| `order-cancelled` | `OrderCancelled{order_id,user_id,customer_id?,total_cents,item_count}` | order outbox | notification (`notification`), payment (`payment-refund`) |
| `order-stock-release` | `OrderStockRelease{order_id}` | order outbox (cancel / payment failed) | catalog (`catalog-stock`) |
| `payment-initiated` | `PaymentInitiated{payment_id,order_id,user_id,amount_cents,currency}` | payment outbox | **none** |
| `payment-succeeded` | `PaymentSucceeded{...,provider_ref}` | payment outbox | order (`order-payment`), notification (`notification`) |
| `payment-failed` | `PaymentFailed{...,reason}` | payment outbox | order (`order-payment`), notification (`notification`) |
| `payment-unknown` | `PaymentUnknown{...,reason}` | payment outbox | order (`order-payment`) |
| `payment-refunded` | `PaymentRefunded{payment_id,order_id,user_id,amount_cents,currency,provider_ref,reason}` | payment outbox | **none** |
| `order-paid` | `OrderPaid{order_id,user_id,customer_id,total_cents,shipping_quote_id,shipping_method_id,destination_snapshot}` | order outbox | shipping (`shipping-shipment-creation`) |
| `review-created` | `ReviewCreated{review_id,created_at}` | catalog outbox | catalog (`review-moderation`) |
| `review-moderated` | `ReviewModerated{review_id,product_id,decision,rating}` | catalog outbox (moderation transaction) | catalog (`review-rating-aggregator`) |
| `shipment-created` | `ShipmentCreated{shipment_id,order_id,user_id,method_id,shipping_cost_cents,destination_snapshot,created_at}` | shipping outbox | notification (`notification-shipment`) |
| `shipment-shipped` | `ShipmentShipped{shipment_id,order_id,user_id,tracking_code,carrier,shipped_at}` | shipping outbox | notification (`notification-shipment`) |
| `shipment-delivered` | `ShipmentDelivered{shipment_id,order_id,user_id,delivered_at}` | shipping outbox | notification (`notification-shipment`) |
| `notification-email` | delivery id (JSON string) | notification delivery relay | notification (`notification-email`) |
| `notification-sms` | delivery id (JSON string) | notification delivery relay | notification (`notification-sms`) |
| `notification-push` | delivery id (JSON string) | notification delivery relay | notification (`notification-push`) |

Rules:

- `order`, `payment`, `catalog`, and `shipping` publish; all four write the outbox row in the same transaction as the aggregate change. Shipping now writes its outbox row in the same transaction as the shipment/quote (`publisher.Send(ctx, msg, tx)` with a valid `pgx.Tx`). The `notification` delivery relay bypasses the outbox and publishes delivery ids straight to the channel topics after claiming a due `notification_deliveries` row.
- `kafka.AllTopics()` is the canonical list of the 17 source topics; compose provisions a `<topic>.dlq` for each, and a test asserts the two stay in sync.
- Notifying payloads must carry `user_id` (and `order_id` for idempotency). Notification types are `order_confirmation`, `order_cancelled`, `payment_succeeded`, `payment_failed`, `shipment_created`, `shipment_shipped`, and `shipment_delivered`; all are `transactional` and records are unique on `dedupe_key` (`order_id/type`).
- Ingest resolves contacts (email via the auth contract, phone via the customer default address; push tokens are not stored yet), applies the preference gate, and stores the notification plus one `PENDING` `notification_deliveries` row per eligible channel without calling a provider. The relay publishes each due delivery id to `notification-email|sms|push`; the channel consumer renders the template, calls the provider, and marks the delivery `SENT`/`DELIVERED`, retrying transient errors with backoff and dead-lettering permanent or exhausted failures to `<channel-topic>.dlq`. A redelivered event reuses the existing notification and its deliveries.
- Internal plumbing events (`order-stock-release`, `payment-initiated`, `payment-unknown`, `payment-refunded`, `review-created`, `review-moderated`) produce no notification.
- `payment-unknown` holds stock; `payment-failed` and cancellation emit `order-stock-release`.
- Redelivery is expected: notification dedupes in Mongo, catalog's reservation ledger makes release idempotent, the rating aggregator recomputes from scratch (idempotent), and order treats lost CAS races as no-ops.
- Consumer groups let independent domains each receive every message (order and notification both read payment outcomes; order and notification both read `order-placed`; catalog's moderation and rating-aggregator groups both read the review flow; notification reads all three shipment topics under `notification-shipment` and each channel topic under its own `notification-email|sms|push` group).

## Storage Map

| Store | Data |
| --- | --- |
| Postgres primary/replica | `products`, `product_images`, `product_reviews`, `product_rating_summary`, `stock_reservations`, `categories`, `orders` + items, `payments`, `customers`, `addresses`, `accounts`, `avatars`, `shipping_methods`, `shipping_zones`, `shipping_rates`, `shipping_quotes`, `shipments`, `shipment_tracking_events`, `roles`/`permissions`/`user_roles`/`role_permission`, `outbox` |
| MongoDB | `notifications` (unique `dedupe_key`; also unique `(order_id, type)` for legacy rows), `notification_deliveries` (unique `(notification_id, channel)`), `notification_preferences` (`_id=user_id`), `carts` (unique live cart per customer, versioned), `wishlists` (unique by `customer_id`) |
| Redis | idempotency keys, rate-limit counters, RBAC permission cache (L2), catalog review/rating read cache — all fail-open |
| S3-compatible | product images and avatar binaries; metadata stays in Postgres, reads use presigned URLs |
| Kafka | the topics above; DLQs `<topic>.dlq` provisioned with 14-day retention |

Postgres reads use the replica pool; writes use the primary. sqlc-generated files are never hand-edited (`make -C backend sqlc`); migrations are the schema source of truth through the single consolidated initial migration (`20261106231155_initial_schema.sql`). Compose's `redpanda-init` provisions DLQs for all topics including shipment topics.

## Known API and Event Gaps

- **Shipping reads require authentication.** `GET /v1/shipping/methods` is not public, even though the SPA needs methods to render checkout.
- **Quote pricing is advisory until checkout.** `POST /v1/shipping/quotes` accepts an optional `subtotal_cents` and prices the quote with the rate's free-shipping rule, but the amount charged is decided at checkout: the cart re-prices the quote with the subtotal it computed from its own items, so a customer cannot buy free shipping by claiming a smaller cart.
- **Quote and shipment reads are not owner-scoped.** `GET /v1/shipping/quotes/{id}`, `GET /v1/shipping/shipments/{id}`, and `GET /v1/shipping/shipments` return another user's data to any authenticated `shipping:read` holder; the shipment list even accepts an arbitrary `user_id` query override. Only the tracking-history read (`/events`) checks shipment ownership.
- **Review summary has no endpoint.** `GetReviewSummary` and `GetRatingSummary` exist as application use cases in the catalog domain but are not exposed over HTTP; clients read the denormalized `total_rating`/`total_reviews` counters on the showcase product DTO.
- **Payment reconciliation and quote expiry have no endpoint.** Both are background/application capabilities (`payment` reconciler in the worker, `shipping` `ExpireQuotes` use case) with no HTTP route — and `ExpireQuotes` is not attached to any scheduler, so expired quotes are only reaped when the use case is called.
- **Shipping admin/lifecycle endpoints have no SPA UI.** The API exposes method/zone/rate CRUD and shipment process/dispatch/deliver/tracking-event routes, but the Angular app has no component that calls them.

## Observability

- Prometheus metrics cover HTTP requests/duration, cache hit/miss, idempotency outcomes, rate limiting (blocked count, remaining tokens), authz denials, auth events, domain use cases, Kafka publish/consume latency, consumer retries/restarts, dead-letter routing by disposition, consumer lag, backpressure waits/paused state, DLQ replay parking, outbox records, DLQ messages, and Redis command latency/throughput/availability/pool capacity. Review counters currently reuse the `productUpdated` metric (`IncReviewAdded`/`IncReviewModerated` are aliases) — there are no dedicated review metrics yet. Per-user request rates are intentionally not exported as labels (cardinality); use the structured access logs (which include the authenticated user id) for per-user analysis.
- Domain logs use `log/slog` with `*Context`; HTTP/gateway middleware log method, path, status, duration, remote IP, latency, and bytes, plus the `X-Request-ID` correlation id.
- The gateway and server expose `/metrics`; server and worker `readyz` both check Postgres (read/write), Mongo, and Redis (Kafka and S3 are not probed); the gateway's `readyz` is dependency-free.
