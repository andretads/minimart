# Domain Behaviors

## Domain Map

| Domain | Role | Storage | Contract / events |
| --- | --- | --- | --- |
| `auth` | accounts, password hashing, JWT/cookies, RBAC | Postgres (+ Redis permission cache) | publishes `AuthClient`; no events |
| `customer` | customer records keyed by user id | Postgres | publishes `CustomerClient`; no events |
| `address` | customer shipping addresses | Postgres | publishes `AddressClient`; no events |
| `category` | product categories | Postgres | publishes `CategoryClient`; no events |
| `catalog` | products, stock reservations, reviews + moderation, rating aggregate | Postgres + S3 + moderation provider + Redis read cache | publishes `CatalogClient`; consumes `order-stock-release`, `review-created`, and `review-moderated`; outbox |
| `cart` | shopping cart and checkout orchestration | MongoDB | consumes catalog/customer/order/shipping; no events |
| `wishlist` | user wishlist | MongoDB | consumes catalog/customer; no events |
| `order` | order lifecycle, stock reservation, shipping snapshot | Postgres | publishes `OrderClient`; consumes `order-placed` and payment outcomes; outbox |
| `payment` | payment attempts via fake provider | Postgres | consumes none; outbox |
| `shipping` | methods, zones, rates, quotes, shipments, tracking | Postgres | publishes `ShippingClient`; consumes `order-paid`; outbox |
| `notification` | user notifications, per-channel deliveries, and preferences | MongoDB | consumes seven user-facing topics (order, payment, shipment) plus the three delivery topics; publishes delivery ids to `notification-email|sms|push` via its worker relay (no outbox) |
| `showcase` | public product read model | Postgres replica | publishes none |
| `avatar` | account avatar metadata + S3 object | Postgres + S3 | publishes none |

**Aggregate ownership**: only the owning module writes its tables. Cross-domain effects are synchronous contracts (`category←catalog`, `customer←auth/order/cart/address/shipping/catalog`, `catalog←order/cart/wishlist`, `order←cart`, `address←shipping`, `shipping←cart`, `auth←notification`) or Kafka events. Nothing writes another domain's tables.

## Auth

- **Account** is valid by construction via `NewAccount(id, username, password, email)`: username trimmed, 1–200 chars; password 6–72 bytes; email trimmed, valid (`@`, non-space local/domain, dotted domain), ≤254; password stored as bcrypt `PasswordHash`. `CreatedAt`/`UpdatedAt` are UTC.
- **Signup** creates the account, then calls the customer contract `EnsureCustomer` and assigns the default `customer` role. A duplicate username/email or weak/invalid input fails before or at persistence. Signup does **not** log the user in.
- **Login** verifies the bcrypt hash, issues an access JWT and a refresh JWT, encrypts them into `mm_at`/`mm_rt`, sets both cookies, and returns user information only. Passwords are never logged.
- **Sessions**: access token 15m; refresh 7d, or 30d when `remember_me` is true **and** `AUTH_REMEMBER_ME_REFRESH_TOKEN_TTL > 0`. Refresh performs sliding renewal: it issues both a new access cookie and a new refresh cookie, preserving the remember-me lifetime. Logout clears both cookies. Changing the password does not revoke existing tokens.
- **Change password**: self only (account id from claims); confirmation must match; current password must verify; new password follows the same 6–72 rule; only the hash is updated.
- **RBAC**: roles are `customer` (default), `support`, `admin`. Permissions are seeded by migrations and cached in Redis per account for `RBAC_PERMISSIONS_CACHE_TTL` (default 5m). `ADMIN_USERNAMES` grants `admin` idempotently at boot through `EnsureAdmins`.
- **Seeded permission behavior**: customers get catalog/order/payment read-write for transactional flows, `customers:read`, `customers:write`, notification read/write, address read/write, avatar read/write, `shipping:read`, `reviews:write`, and account write. They do **not** get `catalog:write` or shipping admin writes. Support adds broader reads; admin adds everything including catalog and shipping admin writes.
- **Session caveats**: there is no token revocation list and no CSRF token; `SameSite=Lax` is the CSRF mitigation. Refresh performs sliding renewal and preserves the remember-me lifetime. `/v1/auth/logout` is public at both authz and auth middleware, so it clears cookies even with an expired access token.

## Customer

- `Customer{ID, UserID, DisplayName, Status, CreatedAt, UpdatedAt, Version, DeactivatedAt}`. `NewCustomer` requires a non-empty `UserID`; display name is trimmed ≤200; status starts `active`; version 1.
- `EnsureCustomer(userID, displayName?)` is get-or-create keyed by `user_id`, used at signup, order placement, cart checkout, and address operations. Concurrent creates rely on the unique `user_id` index; the loser re-reads.
- `UpdateDisplayName` trims and enforces ≤200 and does not mutate `Version`; storage bumps it. `Deactivate` is idempotent, preserves the first `DeactivatedAt`, and is a bump-only system write.
- Status is `active`/`deactivated`; there is no reactivation method. Reading a deactivated customer still returns the record.
- Customer creation HTTP is protected by `customers:write`; customer reads are protected by `customers:read`, which the default `customer` role holds via the initial migration (`20261106231155_initial_schema.sql`), so the profile's own customer fetch succeeds.

## Address

- An address belongs to exactly one customer. `NewAddress` validates: customer id required; label ≤50; full name ≤100; street ≤200; city ≤100; state ≤100; zip ≤20; country ≤100; phone ≤30. All strings are trimmed; `CreatedAt`/`UpdatedAt` UTC; version starts at 1. `country` is normalized through `pkg/country` on create and update: known localized names (`brasil`, `brazil`, `brasil.`) become `BR` and any two-letter value is upper-cased, but **any other value is stored as-is** — the domain does not guarantee an ISO-3166 alpha-2 code.
- `UpdateDetails` validates the same fields and updates `UpdatedAt`, not `Version`. `SetDefault`/`UnsetDefault` only flip the flag when needed.
- **One default per customer** is enforced by `addresses_customer_default_unique`. Creating a second default, or updating another address to default, conflicts (409). `SetDefault` clears the previous default in one repository operation.
- Application use cases take the **customer id resolved from the access cookie**, never from the request. Get/update/delete/set-default verify `address.CustomerID == callerCustomerID` and return `ErrAddressNotFound` on mismatch (no existence leak).
- Update/delete require a positive `version`; stale version is `409`, missing is `404`. Deletion uses `(id, customer_id)`.
- List is scoped to the caller's customer. Address data is never read or written by order; orders are not currently linked to an address id.

## Category

- `Category{ID, Name, Icon, CreatedAt, UpdatedAt, Version}`. `NewCategory` trims the name (required, ≤100) and icon (optional, ≤500), stamps UTC timestamps, version 1. `UpdateNameAndIcon` validates and updates `UpdatedAt` only; storage bumps `Version`.
- Names are unique **case-insensitively** (`categories_name_lower_idx`). Duplicate create/update is `ErrCategoryExists` (409).
- Update/delete use optimistic locking on `version`; stale → 409, missing → 404. Delete requires a positive version. Products reference categories by FK, so deleting a referenced category fails at the database level rather than cascading.
- Categories are consumed by catalog on product create/update; `GetCategory` validates existence before the product is constructed/updated. Category deletion does not coordinate with catalog beyond the FK.
- Category endpoints are protected: `GET /v1/categories` and `/v1/categories/{id}` require `catalog:read`; create/update/delete require `catalog:write`. Public category browsing is available through `GET /v1/showcase/categories`.

## Catalog

### Products

- `Product{ID, Name, PriceCents, Stock, Version, CreatedAt, UpdatedAt, CategoryID, SKU, Description, Details, TotalRating, TotalReviews}`. Money is integer cents only.
- `NewProduct` requires trimmed non-empty name, `PriceCents > 0`, `Stock >= 0`, non-empty `CategoryID`; SKU is optional but, when provided, trimmed non-empty. Version 1; UTC timestamps. `TotalRating`/`TotalReviews` start at 0.
- Updates (`UpdateNameAndPrice`, `UpdateCategory`, `UpdateSKU`, `UpdateDescription`, `UpdateDetails`) validate their field and leave version/timestamps to storage. Update does **not** change stock.
- Product writes require a category that exists via `category.GetCategory`; missing category → `ErrProductNotFound` from the contract mapped to a 400. SKU uniqueness is enforced by the schema; duplicate → 400.
- Checked updates require the caller's version (`If-Match` wins). Storage CAS returns the new version; stale → 409, missing → 404. GET returns `ETag`.
- `ProductImage{ID, ProductID, ObjectKey, Bucket, ContentType, SizeBytes, Position, Version, CreatedAt, UpdatedAt}`. A product has zero or more images; bytes live in S3. Upload accepts JPEG/PNG/WebP/GIF up to the configured limit. Position is assigned by storage (`MAX(position)+1`), and the lowest position is the primary image. Adding or deleting an image bumps the owning product version in the same transaction. Product create/update no longer accept image fields; images are managed through `POST/GET/DELETE /v1/products/{id}/images`.
- Category filtering is supported on product list and showcase.

### Reviews

- `Review{ID, ProductID, CustomerID, Description, Rating, Status, CreatedAt}` plus moderation fields (`ModeratedAt`, `ModerationDecision`, `ModerationConfidence`, `ModerationCategories`, `ModerationExplanation`, `ModerationModel`, `ModerationModelVersion`). `AddReview` requires rating 1–5, a trimmed non-empty description, a non-empty customer id, and an existing product.
- Reviews are created `pending`; allowed statuses are `pending`, `approved`, `rejected`, `needs_review`. `GetReviews` returns **only `approved`** rows (the SQL filters on `status = 'approved'`). A customer may post **one review per product**: the unique index `uq_product_reviews_product_customer` on `(product_id, customer_id)` turns a second review into `ErrDuplicateReview` (409).
- `AddReview` inserts the review and a `review-created` outbox row in the same transaction. The catalog worker consumes `review-created` (`review-moderation` group), calls `ModerationService.ModerateReview`, and persists `approved`/`rejected`/`needs_review` with confidence, categories, explanation, model, and model version, writing a second `review-moderated` outbox row in the same transaction. A provider error (Ollama unreachable) aborts moderation and leaves the review `pending`; an invalid provider decision/confidence also leaves it `pending`; moderating an already-moderated review returns `ErrReviewAlreadyModerated`.
- **Rating aggregate**: two paths keep the counters in sync. (a) On approval the moderation service recomputes the sum/count from the approved reviews and writes `products.total_rating`/`products.total_reviews` (both `BIGINT`) plus the `product_rating_summary` row (`total_reviews`, `rating_sum`, `average_rating`). (b) A second catalog consumer group (`review-rating-aggregator`) consumes `review-moderated` and calls `Service.UpdateRatingSummary`, which recomputes the same values idempotently. `GetRatingSummary` reads the summary row (missing row ⇒ zero summary); `GetReviewSummary` derives the average from the product counters. **Neither has an HTTP route** — the public counters travel on the showcase product DTO.
- Review reads are paginated: `ListReviews` (page ≥1, page size 1–100) returns approved reviews newest-first plus `Total`/`TotalPages`. `GetReviewsPaginated` is cached in Redis under `catalog:reviews:<product_id>:<limit>:<offset>` (1m TTL); the rating summary is cached under `catalog:rating:<product_id>` (1m TTL) and that key is deleted when a review is saved. The review-page cache is **not** invalidated on save, so a freshly approved review can take up to a minute to appear in a cached page.
- With `MODERATION_ENABLED=false` the noop provider approves every review (model `noop`); with the provider down, reviews stay `pending` and invisible publicly.
- There is no HTTP moderation endpoint; moderation is worker-driven. Rating/description validation reuses `ErrInvalidRating`/`ErrEmptyName`; a missing customer is reported as `ErrEmptyCustomerID` (400).
- `POST /v1/products/{id}/reviews` requires the `reviews:write` permission (held by the `customer` and `admin` roles). The handler resolves the caller from the session claims through the customer contract (`EnsureCustomer`) and attributes the review to that customer, so the customer id is never taken from the request body. `GET /v1/products/{id}/reviews` requires `catalog:read`.

### Stock

- Reservation is keyed by order id. `ReserveStock(ctx, orderID, items)` is all-or-nothing in one transaction: each product is decremented with `UPDATE ... WHERE id=$1 AND stock >= $q`; 0 rows → sold out/missing. A `stock_reservations` ledger keyed `(order_id, product_id)` makes a repeated reserve a no-op.
- `ReleaseStock(ctx, orderID)` marks outstanding reservations released and restores product stock in one transaction; a redelivered release writes nothing. Reserve/restore are bump-only writes (version increments without a caller check).
- The order domain reserves before saving; if saving fails it releases best-effort. Cancellation/payment failure emits `order-stock-release`, which the catalog consumes.

## Cart

- `Cart{ID, CustomerID, Status, CheckoutKey, OrderID, Items, CreatedAt, UpdatedAt, Version}` in MongoDB. Items snapshots name/unit price at add time; `EstimatedTotalCents` is a snapshot total, not an authoritative order total.
- Status: `active` → `checkout_pending` → `consumed`; `consumed` is terminal. `BeginCheckout(key)` requires `active`; `MarkConsumed(orderID)` and `ReleaseCheckout()` require `checkout_pending`. Item mutations (`AddItem`, `UpdateItemQuantity`, `RemoveItem`, `Clear`) require `active`; quantity must be ≥1. Adding an existing product increments its quantity.
- A customer has at most one live cart (`active` or `checkout_pending`) via a unique partial Mongo index. `GetOrCreateCart` creates a new active cart when none exists; consumed carts can be read but never reactivated.
- `Update(ctx, cart, expectedVersion)` is CAS on `(_id, version)`, setting `version = expected + 1`; a 0-match probe distinguishes 404 from 409.
- **Checkout**: requires `user_id`, an `Idempotency-Key`, a `shipping_quote_id`, and an `address_id`; `EnsureCustomer` → `GetQuote` + customer ownership check (`ErrQuoteOwnershipMismatch`) + `ValidateQuote` → load live cart (or consumed checked for replay) → reject empty → `BeginCheckout` + CAS → re-read for the new version → `PriceQuote` the quote with `EstimatedTotalCents()` (the subtotal of the frozen cart) → `PlaceOrder(userID, items, quote snapshot)` → `MarkConsumed` + CAS. The order receives the quote's method id, destination snapshot, and the re-priced shipping cost; `address_id` is required but not separately re-validated by cart.
  - The quote the customer picked was priced against a subtotal they supplied; the checkout re-prices it with the cart's own subtotal, so the free-shipping threshold is applied to the real cart total. A pricing failure releases the cart back to `active`.
  - A checkout that already started (cart `checkout_pending` or `consumed`) is not re-priced: it reuses the stored quote cost so the order's idempotency fingerprint stays stable across retries.
  - A business failure (insufficient stock, unknown product, no items, invalid quantity, idempotency conflict) releases the cart back to `active`.
  - A technical failure leaves it `checkout_pending`; retry with the **same** key replays the order through order idempotency.
  - If marking consumed fails after a successful order, the order is still returned (non-fatal); a later retry returns the same order.
  - Concurrent checkout with a different key while `checkout_pending` is rejected.
- The endpoints are authenticated (`cart:read`/`cart:write`) and scoped to the caller from the access-token cookie.

## Wishlist

- `Wishlist{ID, CustomerID, Items, CreatedAt, UpdatedAt}` in MongoDB. One wishlist per customer.
- Items are snapshots with `ProductID` and `AddedAt`. Supports add/remove/clear; prevents duplicates.
- `NewWishlist` requires non-empty `CustomerID`. Uses catalog contract to validate product exists on add.
- Endpoints are authenticated (`wishlist:read`/`wishlist:write`) and scoped to the caller from the access-token cookie via `customer.EnsureCustomer`. Uses customer contract for validation.

## Order

- `Order{ID, UserID, CustomerID, IdempotencyKey, Items, TotalCents, Status, CancelledAt, CreatedAt, ShippingQuoteID, ShippingMethodID, ShippingMethodName, ShippingCostCents, DestinationSnapshot, events}`. Item names/prices are snapshots; `NewOrder` requires ≥1 item, positive quantities, computes line and order totals, starts `placed`, and records `OrderPlaced`. Shipping fields are optional on the domain object: `PlaceOrder` adds the quote id, method id, method name, destination snapshot, and shipping cost to `TotalCents` when both quote and method ids are supplied.
- **Shipping persistence**: the `orders` table carries the shipping snapshot (`shipping_quote_id`, `shipping_method_id`, `shipping_method_name`, `shipping_cost_cents`, `destination_snapshot` JSONB, added by the initial migration (`20261106231155_initial_schema.sql`). The writer stores it with the order and the reader returns it, so `GetOrder`/`GetByID`/`ListByUserID`, the HTTP order response, and the order contract all expose the shipping data. An order placed without shipping stores empty values.
- Status machine:
  - `placed` → `paid` | `payment_failed` | `payment_unknown` | `cancelled`.
  - `payment_unknown` → `paid` | `payment_failed`.
  - `paid`, `payment_failed`, and `cancelled` are absorbing.
- `MarkPaid`: no-op if already paid, cancelled, or payment_failed; records no event.
- `MarkPaidWithShipping`: same guards as `MarkPaid`, but records `OrderPaidEvent` carrying `order_id`, `user_id`, `customer_id`, `total_cents`, the order's own shipping quote/method ids, and the destination snapshot. The event can never disagree with the persisted snapshot because the aggregate supplies it.
- `MarkPaymentFailed`: applies only from `placed`/`payment_unknown`; records `OrderStockRelease`. A late success cannot resurrect a failed order — retry is a new order.
- `MarkPaymentUnknown`: applies only from `placed`; holds stock; no event.
- `Cancel`: rejects `paid` (`ErrOrderNotCancellable`) and already-cancelled (`ErrOrderAlreadyCancelled`); from placed/unknown it sets `CancelledAt` and records `OrderCancelled` + `OrderStockRelease`.
- `AutoCancelAfter`: placed orders are eligible for automatic cancellation after 5 minutes (`domain.AutoCancelAfter`). The worker (`application.AutoCanceller`) scans placed orders created at or before now minus 5 minutes in batches of `ORDER_AUTO_CANCEL_BATCH_SIZE` (default 100) every `ORDER_AUTO_CANCEL_INTERVAL` (default 15s), cancels each one through `Cancel()` + CAS `Update(..., StatusPlaced)`, and counts successes. Concurrent status changes are skipped without clobbering. A placed order's HTTP response carries `auto_cancel_at` (creation time plus 5 minutes).
- **Placement** resolves prices/names through catalog `GetProduct` (client-supplied prices are never trusted), `EnsureCustomer`, constructs the order, `ReserveStock`, then saves order + items + outbox in one transaction. Save failure triggers a best-effort stock release.
- **Idempotency**: Redis lookup/reserve in front, unique `orders.idempotency_key` authoritative; same key/same fingerprint replays, different fingerprint is 409, concurrent duplicate is in-progress. Placement is idempotent end to end; a replay returns the stored order.
- **CAS status writes**: `Update` conditions on the status read before the change. A lost race is benign in `ApplyPaymentOutcome`; a stale cancel reports `ErrOrderStateChanged`/not found.
- **Payment outcome consumer**: `payment-succeeded|failed|unknown` under `order-payment` maps to `ApplyPaymentOutcome` → `Mark*` → CAS. Terminal-event redelivery is a no-op. `payment-failed` releases stock via the outbox.
- **Published contract**: cart places orders synchronously. The contract resolves the customer from `UserID`; transport-neutral errors are translated in `presentation/module/order.go`.
- **Cancellation notification**: `order-cancelled` is consumed by notification; `order-stock-release` is consumed by catalog.

## Payment

- `Payment{ID, OrderID, UserID, IdempotencyKey, AmountCents, Currency, Method, Status, Provider, ProviderRef, FailureReason, CreatedAt, UpdatedAt, events}`. `NewPayment` requires positive amount, non-empty currency/method/user, starts `pending`, and records `PaymentInitiated`.
- Statuses: `pending` → `succeeded` | `failed` | `unknown`; all three are terminal. Applying an outcome to a non-pending payment is `ErrPaymentNotPending`. `Refund` moves `pending`, `succeeded`, or `unknown` → `refunded` (terminal) and records `PaymentRefunded`; `failed` and `refunded` are not refundable (`ErrPaymentNotRefundable`). A `Refund` result must be `approved` (`ErrInvalidTransition` otherwise); the refund keeps the payment's charge reference when the provider echoes none and clears `FailureReason`.
- Provider mapping (`ApplyOutcome`): `approved` → `succeeded` + `PaymentSucceeded`; `declined` → `failed` + `PaymentFailed`; `technical_error`/`timeout` → `unknown` + `PaymentUnknown`. A timeout is **never** recorded as failed because the charge may have succeeded.
- Provider port is `Authorize`, `Capture`, `GetStatus`, `Refund`, `Name`. The fake provider approves unless the amount matches configured decline/timeout amounts; its refund approves and echoes the charge reference.
- **Refunds reverse captured charges on cancelled orders.** Three paths converge: (1) the `order-cancelled` consumer (worker, `payment-refund` group) lists the order's `succeeded` payments (`repo.ListSucceededByOrderID`) and refunds each — at-least-once delivery makes an already-refunded payment or a lost race (`ErrPaymentNotRefundable`) a no-op; (2) `CreatePayment` re-checks the order after capture (`chargeLandsOnCancelledOrder`) and refunds inline with reason "pedido cancelado" — when that refund fails the charge reference is persisted on the still-`pending` payment via `repo.Update` so the reconciler can retry, and the request fails with `ErrRefundFailed`; (3) the reconciler refunds a captured charge it confirms on a cancelled order instead of finalizing it. `POST /v1/payments/{id}/refund` refunds a `succeeded` payment manually with reason "reembolso solicitado", but only when its order is `cancelled` (`ErrOrderNotCancelled`); an already-refunded payment replays. `refunded` is a CAS transition (`repo.Refund`, `WHERE status IN ('pending','succeeded','unknown')`).
- **Reconciliation** (worker only): `application.Reconciler` runs on `PAYMENT_RECONCILE_INTERVAL` (1m) and asks the provider for the status of `pending`/`unknown` payments older than `PAYMENT_RECONCILE_AGE` (5m), limited to `PAYMENT_RECONCILE_BATCH_SIZE` (100). It lists with `repo.ListReconcilable`, calls `provider.GetStatus(providerRef)`, applies the outcome with `Payment.ApplyReconciliation`, and persists it with `repo.Reconcile` (which only transitions a still-pending payment). When the charge is confirmed captured but its order was cancelled (possible because `Order.Cancel` allows cancelling from `payment_unknown`), the reconciler refunds instead; the order lookup fails open (finalize normally, leave the refund to a retry or the manual endpoint). Payments without a `provider_ref` are skipped; provider errors, non-pending payments, and failed writes are logged and skipped without stopping the loop. `GetStatus` therefore has a real caller now. There is **no HTTP endpoint** for reconciliation.
- **Creation verifies the referenced order before charging.** `CreatePayment` loads the order through the order contract and rejects a missing order (`ErrOrderNotFound`, 404) and, as 400s, a `user_id` that is not the order owner (`ErrPaymentUserMismatch`), an `amount_cents` that differs from the order total (`ErrPaymentAmountMismatch`), and a mismatched non-empty order currency (`ErrPaymentCurrencyMismatch`). The order domain currently carries no currency, so the currency check normally does not fire. The payment is then charged for the validated amount. Idempotency is Redis + unique `payments.idempotency_key`; a replay returns the stored payment, a different fingerprint 409s. History is outbox only.
- **Payment guard**: `CreatePayment` also rejects `cancelled` and `payment_failed` orders with `ErrOrderNotPayable` (409).
- Events: `payment-initiated` (no consumer), `payment-succeeded`/`payment-failed` (order + notification), `payment-unknown` (order only), `payment-refunded` (no consumer; carries `provider_ref` and `reason`). `user_id` rides on every event for notification/attribution.
- Payment does not write orders; the order domain reacts to events and applies its own CAS status transition. Payment **consumes** `order-cancelled` (under `payment-refund`) to drive refunds.

## Shipping

- **Methods** (`ShippingMethod{ID, Name, Description, Active, EstimatedMinDays, EstimatedMaxDays, ...}`): admin CRUD; inactive methods cannot be quoted. Name/description bounded; day estimates non-negative.
- **Zones** (`ShippingZone{ID, Name, Country, State, City, PostalCodeStart, PostalCodeEnd, Active, ...}`): admin CRUD. `MatchesDestination(country, state, city, zip)` matches the zone's configured fields; the first matching zone wins. Empty fields act as wildcards; postal ranges compare lexically. Country comparison goes through `pkg/country.Equal`, so both sides are normalized (aliases → alpha-2, two-letter values upper-cased): a zone configured for "BR" matches an address stored as "Brasil" or "br".
- **Rates** (`ShippingRate{ID, ZoneID, MethodID, BaseCostCents, FreeThresholdCents, Active, ...}`): admin CRUD. A quote picks the first active rate for the matched zone/method. Money is integer cents; `CalculateCost(subtotalCents)` returns `0` when the subtotal reaches `FreeThresholdCents` (threshold `0` disables free shipping) and the base cost otherwise.
- **Quotes** (`ShippingQuote{ID, CustomerID, MethodID, AddressID, IdempotencyKey, DestinationSnapshot, ShippingCostCents, SubtotalCents, FreeThresholdCents, EstimatedMinDays, EstimatedMaxDays, Status, ExpiresAt, ...}`): statuses `active`, `expired`, `consumed`. Creation requires a customer, a non-empty `Idempotency-Key`, an active method, and an address owned by the customer (via the address contract); it snapshots the destination, the subtotal it was priced against, and the rate's free threshold, and expires after the configured TTL. `ValidateQuote` rejects expired/consumed quotes and enforces ownership. Quotes are single-use: shipment creation consumes the quote. The `shipping_quotes` table keeps the pricing snapshot (`subtotal_cents`, `free_threshold_cents`) and the idempotency fields (`address_id`, `idempotency_key`), all in the initial migration (`20261106231155_initial_schema.sql`).
- **Quote creation is idempotent.** A repeated request with the same key and the same customer/method/address/subtotal replays the original quote (HTTP 201 with the same quote); the same key with any changed parameter is `ErrIdempotencyConflict` (409); a concurrent duplicate with an in-flight key is `ErrQuoteInProgress` (409). Redis (`shipping:quote:<client key>`) is the fast path; `shipping_quotes.idempotency_key` (partial unique index, non-empty keys only) is authoritative, so idempotency survives an empty/unavailable Redis. A missing key is `ErrIdempotencyKeyRequired` (400); the HTTP layer also rejects it before calling the service.
- **Pricing is priced twice, and the second price wins.** Quote creation accepts an advisory `SubtotalCents` and prices the quote with it; the cart checkout re-prices the quote with the subtotal it computed from its own items (`PriceQuote` in the published contract) before placing the order, so the amount charged never depends on what the customer claimed. The re-priced cost is persisted on the quote, so the shipment created from the order-paid event carries the same cost as the order. A retry of a checkout that already started reuses the stored quote cost to keep the order's idempotency fingerprint stable.
- **Shipments** (`Shipment{ID, OrderID, UserID, MethodID, TrackingCode, Carrier, ShippingCostCents, DestinationSnapshot, Status, ShippedAt, DeliveredAt, ...}`): statuses `pending`, `processing`, `shipped`, `delivered`, `failed`. One shipment per order (`shipments.order_id` unique). `CreateShipmentFromOrderPaid` is idempotent by `order_id`: the Redis key `shipping:shipment:<order_id>` is the fast path, the unique `shipments.order_id` index is authoritative, an existing shipment is returned, a key reused with a different `order-paid` payload is `ErrIdempotencyConflict`; otherwise it validates the quote (ownership + `active`), validates the method is active, creates a `pending` shipment using the quote's cost/destination, and consumes the quote.
- **Fulfillment**: `ProcessShipment` moves `pending` → `processing`; `DispatchShipment` moves `processing` → `shipped` and records tracking code/carrier + `shipped_at`; `DeliverShipment` moves `shipped` → `delivered` + `delivered_at`. Invalid transitions return `ErrInvalidTransition`.
- **Quote expiry**: `Service.ExpireQuotes(ctx)` flips every `active` quote whose `expires_at` has passed to `expired` (one repository statement). It is **not scheduled** — there is no background loop, HTTP route, or worker attachment for it, so expired quotes are only reaped if the use case is invoked.
- **Tracking events** (`TrackingEvent{ID, ShipmentID, Status, Description, Location, OccurredAt, CreatedAt}`) are append-only; adding one does not change the shipment status. Reads are available through the shipment HTTP routes.
- **Ownership**: quote/shipment **writes** and the tracking-history read are ownership-checked (quote creation resolves an address owned by the caller; shipment creation validates quote ownership; `GET /v1/shipping/shipments/{id}/events` compares the shipment's `UserID` with the caller and returns 403 on mismatch). `GET /v1/shipping/quotes/{id}`, `GET /v1/shipping/shipments/{id}`, and `GET /v1/shipping/shipments` are **not** owner-scoped: any authenticated `shipping:read` holder can read another user's quote/shipment, and the list accepts an arbitrary `user_id` override. A shipment lookup by `order_id` returns a single-item paginated list. Admin CRUD/transition routes require the shipping permissions seeded by the initial migration (`20261106231155_initial_schema.sql`).
- **Events**: consumes `order-paid` under `shipping-shipment-creation`; emits `shipment-created`, `shipment-shipped`, and `shipment-delivered` via the transactional outbox (`publisher.Send(ctx, msg, tx)` in the same transaction as shipment/quote). Notification consumes all three under `notification-shipment`.
- **Errors**: quote `ErrQuoteNotFound`/`ErrQuoteExpired`/`ErrQuoteConsumed`/`ErrInvalidQuote`/`ErrQuoteOwnershipMismatch`/`ErrQuoteDestinationMismatch`/`ErrIdempotencyConflict`/`ErrQuoteInProgress`/`ErrIdempotencyKeyRequired`; method/zone/rate `ErrMethodNotFound`/`ErrMethodInactive`/`ErrZoneNotFound`/`ErrZoneInactive`/`ErrRateNotFound`/`ErrRateInactive`/`ErrNoMatchingRate`; shipment `ErrShipmentNotFound`/`ErrShipmentExists`/`ErrShipmentInProgress`/`ErrInvalidTransition`; address `ErrAddressNotFound`.
- **Tests**: `backend/internal/shipping/test/` contains integration tests (build tag `integration`, `TestMain` via `pkg/tests`) for the Postgres repository/outbox, the `order-paid` Kafka consumer, shipment idempotency, and quote idempotency (replay, DB fallback, conflict, key required).
- **Idempotency**: the module receives an `idempotency.Store` and `SHIPPING_IDEMPOTENCY_TTL` (24h), used by both quote creation and shipment creation; the DB unique indexes are the authoritative fallback. Quote fingerprint = customer/method/address/subtotal; shipment fingerprint = the `order-paid` payload.

## Notification

- Notifications are Mongo rows: `notifications` keeps the audit record (`{ID, Type, UserID, RecipientID, OrderID, TemplateID, Payload, Priority, Status, Subject, Body, SentAt, DeliveredAt, CreatedAt, ReadAt}`); `notification_deliveries` keeps one row per channel per notification (`{ID, NotificationID, Channel, Provider, Status, AttemptCount, NextAttemptAt, LockedBy, LockedUntil, Error}`). Users may mark them read/unread or delete them; the audit payload itself stays immutable.
- Types: `order_confirmation`, `order_cancelled`, `payment_succeeded`, `payment_failed`, `shipment_created`, `shipment_shipped`, `shipment_delivered`, mapping 1:1 to the seven subscribed topics.
- Every notifying payload carries `user_id` and `order_id`. Idempotency is `dedupe_key = order_id/type` (unique index). Ingest resolves the stored notification by that key and reuses it, so a redelivered event writes no second notification and creates no delivery for an unsaved notification.
- **Channels**: `email` (Resend, or a log fallback), `sms` and `push` (log adapters). Contacts come from the auth contract for email and from the customer plus default address for the phone; push tokens are not stored yet.
- **Preferences**: one Mongo doc per user, `_id = user_id`, with the channel toggles `email_enabled`, `sms_enabled`, `push_enabled` and the `marketing_enabled` flag. A missing doc means defaults (email on, marketing off). GET returns defaults; PUT is a full replacement requiring all four booleans and derives the user from claims. Legacy `order_updates`/`promotions`/`security_alerts`/`newsletter` documents are mapped to the new fields on read.
- **Delivery pipeline**: ingest resolves the contacts, applies the preferences, stores the notification plus one `PENDING` delivery per eligible channel, and performs no provider call. The relay claims due deliveries and publishes the delivery id to `notification-email`, `notification-sms` or `notification-push`; the channel consumer renders the template, resolves the contact, calls the provider, and marks the delivery `SENT` (provider accepted) or `DELIVERED` (provider confirmed). The notification status is recomputed from its deliveries.
- **Retries and dead letters**: a transient provider error schedules the next attempt with exponential backoff (bounded by `MaxAttempts`); a permanent error or an exhausted retry budget marks the delivery `FAILED` and returns a `kafka.Permanent` error so the shared consumer dead-letters the delivery id to `<channel-topic>.dlq` with the standard `x-minimart-*` triage headers. A delivery whose notification or delivery row is missing is likewise a permanent Kafka error, so it dead-letters immediately instead of burning the retry budget. Invalid event payloads (empty ids, non-positive totals) are classified permanent at the adapter, while repository/provider outages remain transient.
- **Copy**: subject/body are pt-BR; money is rendered as BRL (`R$ 12.000,00`) via integer cents in `formatMoney`.
- Internal events (`order-stock-release`, `payment-initiated`, `payment-unknown`) produce no notifications.
- Reads are permission-gated and ownership-gated: `GET /v1/notifications` and `GET /v1/notifications/{id}` are scoped to the authenticated user.

## Showcase

- Public read model over the shared `products`, `product_images`, and `product_reviews` tables using the replica only. `Product{ID, Name, PriceCents, CategoryID, Stock, SKU, Description, Details, TotalRating, TotalReviews, Images, CreatedAt}`; no version (timestamps retained for keyset pagination).
- `ListProducts(page_size, cursor?, category_id?)`: keyset pagination, page size 1–100 (default 20), newest-first (`created_at DESC, id DESC`), optional category filter; returns `page_size`, `next_cursor`, `has_more` in `meta` and batch-loads images so each item carries its primary `image_url`. `GetProduct(id)` returns the public view with all images or not-found.
- Image URLs are presigned with `S3_PRESIGN_TTL` in the HTTP layer; the domain keeps raw object keys.
- `ListReviews(productID, page, page_size)` returns approved reviews only, newest first, paginated (page ≥1, page size 1–100, default 20) with `Total`/`TotalPages`. Product rating counters are updated when moderation approves a review, so they only reflect moderated reviews.
- Read-only: no contract, no Kafka, no writes, no auth, no ETag.

## Avatar

- `Avatar{AccountID, ObjectKey, Bucket, ContentType, SizeBytes, Version, CreatedAt, UpdatedAt}`; one row per account. `NewAvatar` requires non-empty account/object key/bucket, a supported content type (JPEG/PNG/WebP/GIF), and positive size.
- Upload is self-only: the path `{account_id}` must equal the authenticated account id. Validates content type and size (`S3_MAX_UPLOAD_SIZE_MB`), uploads to the deterministic key `avatars/{accountID}/avatar`, upserts metadata (`ON CONFLICT (account_id)`, preserving `created_at`, incrementing `version`), and returns a presigned URL (`S3_PRESIGN_TTL`). Re-upload overwrites the same S3 object. If the metadata save fails after upload, the uploaded object is deleted as best-effort cleanup; if presigning fails, the request fails instead of returning an empty URL.
- Read returns metadata plus a presigned URL; it requires `avatars:read` but is not self-only.
- The database write and S3 upload are not transactional: if the DB save fails after upload, the uploaded object is cleaned up best-effort; if S3 presigning fails, the request fails. Product-image delete treats the database as authoritative and logs (but does not fail on) S3 cleanup errors.

## Cross-Domain Flows

1. **Signup** → auth account → `EnsureCustomer` → default role. Customer `user_id` is the account id.
2. **Add to cart** → catalog `GetProduct` for name/price snapshot → ensure customer → Mongo live cart.
3. **Shipping quote** → authenticated user + `method_id` + `address_id` + `subtotal_cents` + `Idempotency-Key` → customer contract resolves customer → address contract validates ownership → zone match → active rate → idempotent quote snapshot with expiry.
4. **Checkout** → cart validates the quote (ownership + active) → cart begins checkout (CAS) → order `PlaceOrder` (idempotent, stock reserved, shipping snapshot/cost included) → cart consumed, records order id.
5. **Payment** → client-supplied order/user/amount → provider → terminal state + outbox → order consumer applies CAS status on the referenced order; notification consumer records payment notification; failed payment releases stock. When the order carries a shipping snapshot, the consumer marks it paid with shipping, which emits `order-paid`; shipping consumes it to create the shipment with the snapshot's destination.
6. **Cancel** → order emits cancellation + stock release → notification records cancellation, catalog restores stock.
7. **Order placed** → order emits `order-placed` → notification records confirmation; the order domain's own confirmation consumer is a no-op/logging path.
8. **Review** → catalog saves `pending` review + `review-created` outbox row → catalog worker moderates → `approved`/`rejected`/`needs_review`; the moderation transaction also writes a `review-moderated` outbox row → a second catalog consumer group (`review-rating-aggregator`) recomputes the product's rating aggregate → approval updates product counters and makes the review visible in catalog/showcase.
9. **Shipment fulfillment** → shipping transitions `pending` → `processing` → `shipped` → `delivered` and appends tracking events; each transition publishes a shipment event that notification consumes to email the user.
10. **Address/avatar** → auth claims → customer/self checks → own storage. No events.

## Behavioral Invariants and Caveats

- No cross-domain writes; no synchronous call outside published contracts.
- Money is integer cents; client prices are never trusted.
- Updates that matter for concurrency use CAS/optimistic locking and distinguish stale from missing.
- Stock is reserved before order persistence and released exactly once by the ledger/outbox.
- Terminal order/payment/cart states never regress; event redelivery must be harmless.
- Rate/lookup caches never make correctness depend on Redis.
- Categories must exist before products; category naming is case-insensitively unique.
- A customer has one default address and one live cart.
- Notifications are idempotent per `(order_id, type)` and preference-suppressed.
- Shipping quotes are single-use and customer-owned; shipments are one-per-order and idempotent by `order_id`.
- A customer may post at most one review per product; the rating aggregate is recomputed only from approved reviews.
- Known current gaps: shipping reads require authentication; quote/shipment detail reads and the shipment list are not owner-scoped (only the tracking-history read is). The rating summary, payment reconciliation, and shipping quote expiry are worker/application capabilities with no HTTP route; quote expiry is not scheduled at all.
