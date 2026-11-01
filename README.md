# MiniMart

## Project description

**MiniMart** is a sample e-commerce project (a small convenience store). It shows, from start to
end, how a **modular monolith** in Go works with an **Angular SPA** and an **edge gateway**.

The system covers the full store flow: public catalog browsing, sign up and login, cart, checkout,
orders, payments (with a fake provider), shipping, notifications, reviews with AI moderation,
wishlist, and avatars. The code is split into **thirteen domains** — `address`, `auth`, `avatar`,
`cart`, `catalog`, `category`, `customer`, `notification`, `order`, `payment`, `shipping`,
`showcase`, and `wishlist`. They share the process, the database pools, and a contract registry, but
they talk to each other only through published interfaces and events.

The app is shipped as **one Go binary** with five roles (`server`, `worker`, `migrate`, `gateway`,
and `dlq-replay`), plus the Angular SPA built separately. Events that change state are written to a
**transactional outbox** in the same transaction as the data and then published to Kafka at least
once; consumers are idempotent. Critical write requests (order, payment, checkout, and shipping
quote) need the `Idempotency-Key` header. Auth uses **encrypted HttpOnly cookies** with JWT and
RBAC; the SPA never stores tokens.

### Architecture and domains

- **Modular monolith + SPA + gateway**: the Go binary runs the modules; the gateway only does
  proxying, CORS, and edge rate limiting (authorization stays in the app).
- **Roles**: `server` (API), `worker` (outbox, Kafka consumers, and background jobs), `migrate`
  (dbmate), `gateway` (edge proxy), and `dlq-replay` (operator replay of dead-letter queues).
- **Layers per domain**: `domain/` → `application/` → `infrastructure/` and `presentation/`.
- **Calls between domains** go only through `backend/pkg/module/contracts`; adapters live in
  `presentation/module/`.

| Domain | Job | Storage |
| --- | --- | --- |
| `auth` | accounts, passwords, JWT/cookies, and RBAC | Postgres (+ Redis cache) |
| `customer` | customer record per user | Postgres |
| `address` | customer addresses | Postgres |
| `category` | product categories | Postgres |
| `catalog` | products, stock, reviews, and moderation | Postgres + S3 + Redis |
| `cart` | cart and checkout orchestration | MongoDB |
| `wishlist` | user wishlist | MongoDB |
| `order` | order lifecycle | Postgres |
| `payment` | charges through a fake provider | Postgres |
| `shipping` | methods, zones, quotes, and shipments | Postgres |
| `notification` | notifications, per-channel deliveries, and preferences | MongoDB |
| `showcase` | public catalog reads (replica) | Postgres |
| `avatar` | avatar metadata + object in S3 | Postgres + S3 |

### Requirements

For local development:

- **Go 1.27+** and **Docker** (for the local stack and the integration tests).
- **Node.js 22+** and **npm 11.9.0** for the SPA.
- **PostgreSQL 16**, **MongoDB 7**, **Redis 7**, **Kafka** (Redpanda in the local stack), and an
  **S3-compatible** service (local floci/MinIO, AWS S3, or Cloudflare R2).
- Optional: **Ollama** with `llama-guard3:1b` (review moderation), **Resend** (email), **Sentry**,
  and **PostHog** (observability/analytics).

### Build

Backend (from the repo root):

```bash
make -C backend build
```

This creates the `backend/minimart` binary with the version set and, if a `default.pgo` profile
exists, uses PGO. OpenAPI docs are generated in `backend/docs/`.

Frontend:

```bash
cd frontend
npm ci
npm run build
```

The output goes to `frontend/dist/frontend/browser`.

### Configuration

Settings come **only from environment variables**. The `backend/sample.env` file lists all options
and is the reference for local development.

#### General and observability

| Parameter | Description | Example (Default) |
| --------- | ----------- | ----------------- |
| `ENV` | Runtime environment; in `production`, cookies use `Secure` and docs are not served. | `development` |
| `LOG_FORMAT` | Structured log format (`json` or `text`). | `json` |
| `LOG_LEVEL` | Minimum log level (`debug`, `info`, `warn`, `error`). | `info` |
| `SERVER_HTTP_ADDR` | HTTP address for the `server` role. | `:8080` |
| `WORKER_HTTP_ADDR` | Address for `/metrics`, `/livez`, and `/readyz` on the `worker` role. | `:8081` |
| `PROMETHEUS_PORT` / `GRAFANA_PORT` | Ports for the local observability stack. | `9091` / `3000` |

#### Data stores

| Parameter | Description | Example (Default) |
| --------- | ----------- | ----------------- |
| `DATABASE_URL` | Read/write connection to Postgres. | `postgres://minimart:minimart@localhost:5432/minimart?sslmode=disable` |
| `DATABASE_REPLICA_URL` | Read connection to the replica (uses `DATABASE_URL` when empty). | `postgres://...` |
| `MONGO_URL` | MongoDB connection used by `cart`, `wishlist`, and `notification`. | `mongodb://localhost:27017` |
| `MONGO_DATABASE` | MongoDB database name. | `minimart` |
| `REDIS_URL` | Redis for idempotency, rate limit, and permission caches (optional accelerator). | `redis://localhost:6379/0` |

#### Messaging

| Parameter | Description | Example (Default) |
| --------- | ----------- | ----------------- |
| `KAFKA_BROKERS` | Comma-separated Kafka brokers (Redpanda speaks the Kafka protocol). | `localhost:9092` |
| `KAFKA_CONSUMER_MAX_ATTEMPTS` | Tries before the message is sent to `<topic>.dlq`. | `5` |
| `KAFKA_CONSUMER_CONCURRENCY` | Consumer replicas per subscription `(topic, group)`. | `1` |
| `KAFKA_CONSUMER_PAUSE_ENABLED` | Pauses reads when the pools are full (backpressure). | `false` |
| `KAFKA_CONSUMER_LAG_ENABLED` | Turns on the lag monitor per subscription. | `true` |
| `KAFKA_USERNAME` / `KAFKA_PASSWORD` | SASL credentials (needed in the cloud, e.g., Confluent). | `""` |
| `KAFKA_SASL_MECHANISM` / `KAFKA_TLS_ENABLED` | SASL mechanism and TLS for the connection. | `plain` / `false` |

#### Object storage

| Parameter | Description | Example (Default) |
| --------- | ----------- | ----------------- |
| `S3_ENDPOINT` | Internal S3 endpoint (empty for AWS S3). | `http://localhost:9000` |
| `S3_PUBLIC_ENDPOINT` | Endpoint used in browser-facing presigned URLs. | `http://localhost:9000` |
| `S3_BUCKET` | Bucket for product images and avatars. | `minimart` |
| `S3_REGION` | Bucket region. | `us-east-1` |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | Bucket access credentials. | `minioadmin` |
| `S3_USE_PATH_STYLE` | Path-style setting, required for MinIO/floci. | `true` |
| `S3_PRESIGN_TTL` / `S3_MAX_UPLOAD_SIZE_MB` | Presigned URL lifetime and upload size limit. | `1h` / `5` |

#### Auth and authorization

| Parameter | Description | Example (Default) |
| --------- | ----------- | ----------------- |
| `JWT_SECRET` | HS256 secret for JWTs; change it in production. | `<set-a-strong-secret>` |
| `AUTH_ACCESS_TOKEN_TTL` | Access token lifetime (`mm_at` cookie). | `15m` |
| `AUTH_REFRESH_TOKEN_TTL` | Refresh token lifetime (`mm_rt` cookie). | `168h` |
| `AUTH_REMEMBER_ME_REFRESH_TOKEN_TTL` | Refresh lifetime with `remember_me`. | `720h` |
| `COOKIE_ENCRYPTION_KEY` | AES-GCM key, 32 bytes in base64 (`openssl rand -base64 32`). | `""` |
| `COOKIE_DOMAIN` | Parent cookie domain when the SPA and API share a domain. | `.example.com` |
| `ADMIN_USERNAMES` | Users promoted to `admin` at boot; safe to run again. | `<admin-user>` |
| `RBAC_PERMISSIONS_CACHE_TTL` | TTL for the L2 permission cache in Redis. | `5m` |

#### Edge gateway

| Parameter | Description | Example (Default) |
| --------- | ----------- | ----------------- |
| `GATEWAY_HTTP_ADDR` | Gateway HTTP address. | `:8080` |
| `GATEWAY_UPSTREAM` | `server` URL that the gateway proxies to. | `http://localhost:8080` |
| `GATEWAY_TRUSTED_PROXIES` | Trusted proxies for `X-Forwarded-*`. | `""` |
| `GATEWAY_CORS_ORIGINS` | Allowed origins (or `*`). | `*` |
| `GATEWAY_CORS_CREDENTIALS` | Allows credentials in CORS. | `true` |

#### Email, moderation, and business rules

| Parameter | Description | Example (Default) |
| --------- | ----------- | ----------------- |
| `RESEND_API_KEY` | Resend key; without it, emails are only logged. | `""` |
| `RESEND_FROM` / `RESEND_REPLY_TO` | Sender and reply-to for notifications. | `MiniMart <onboarding@resend.dev>` |
| `MODERATION_ENABLED` | Turns on review moderation with Llama Guard. | `true` |
| `MODERATION_ENDPOINT` / `MODERATION_MODEL` | Ollama endpoint and model. | `http://localhost:11434` / `llama-guard3:1b` |
| `MODERATION_TIMEOUT` | Moderation call timeout. | `30s` |
| `PAYMENT_IDEMPOTENCY_TTL` / `ORDER_IDEMPOTENCY_TTL` / `SHIPPING_IDEMPOTENCY_TTL` | How long idempotency keys are kept. | `24h` |
| `PAYMENT_FAKE_DECLINE_AMOUNT_CENTS` / `PAYMENT_FAKE_TIMEOUT_AMOUNT_CENTS` | Amounts that fake a decline and a timeout in the fake provider (`0` disables). | `9999` / `7777` |

#### Frontend (at build time)

| Parameter | Description | Example (Default) |
| --------- | ----------- | ----------------- |
| `API_BASE_URL` | API base URL (empty = same origin). | `https://api.example.com` |
| `SENTRY_DSN` / `POSTHOG_KEY` / `POSTHOG_HOST` | Observability and analytics for the SPA. | `""` / `""` / `https://us.i.posthog.com` |

### Run

#### Local stack

Start the dependencies (Postgres, MongoDB, Redis, floci, Redpanda, gateway, server, worker,
Prometheus, and Grafana):

```bash
make -C backend compose
```

To include Ollama/Llama Guard in the moderation profile:

```bash
make -C backend compose-moderation
```

Apply migrations and, if you want, run the roles outside Docker:

```bash
make -C backend migrate
make -C backend server     # API on :8080
make -C backend worker     # outbox/consumers + metrics on :8081
make -C backend gateway    # optional locally; in compose it listens on :8080 and the server on :8081
```

The SPA dev server proxies `/v1` and `/swagger` to `http://localhost:8080`:

```bash
cd frontend
npm start                  # http://localhost:4200
```

Ports opened by compose: gateway `8080`, server `8081`, worker `8082`, Prometheus `9091`,
Grafana `3000`, Postgres `5432`, MongoDB `27017`, Redis `6379`, floci `9000`, and Redpanda `9092`.

#### Dead-letter queue

DLQ replay is an operator task and runs by hand:

```bash
make -C backend dlq-replay topic=<original-topic> [group=<group>] [dry_run=1]
```

### Quality and tests

```bash
make -C backend fmt
make -C backend lint
make -C backend test
make -C backend test-integration   # testcontainers; needs Docker
make -C backend build

cd frontend
npm run lint
npm run format:check
npm run build
npm run test:ci
```

Behavior specs live in `features/**/*.feature` and the request collection lives in
`backend/collections/minimart.collection.yaml`. Swagger is generated with `make -C backend docs` and
served at `/swagger/` outside production.

### References

- [Go documentation](https://go.dev/doc/)
- [Angular](https://angular.dev/)
- [Docker CLI](https://docs.docker.com/engine/reference/commandline/docker/)
- [PostgreSQL](https://www.postgresql.org/docs/)
- [Apache Kafka](https://kafka.apache.org/documentation/)
- [MongoDB](https://www.mongodb.com/docs/)
- [Redis](https://redis.io/docs/latest/)

### Authors

- **andretads** - [github.com/andretads](https://github.com/andretads)
