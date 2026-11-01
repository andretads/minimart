# Frontend Architecture

Angular SPA in `frontend/`, built and served independently of the Go binary. It calls the MiniMart API through the edge gateway in deployed environments and directly through the dev proxy to the server locally. Authentication is entirely cookie-based; the SPA never receives or stores JWTs.

## Stack & Tooling

- **Angular 21.2** (`@angular/*` 21.2.x, installed 21.2.25), TypeScript `~5.9.2` (5.9.3), Angular CLI/build 21.2.x (installed 21.2.28), npm 11.9.0, RxJS 7.8 (`frontend/package.json`).
- **Standalone components only** — no NgModules. Components declare explicit `imports`; templates and styles are inline (the shell's `app.html`/`app.scss` exist but are empty and unused).
- **Zoneless change detection** (`provideZonelessChangeDetection`) with Angular **signals** (`signal`, `computed`, `effect`) for local and shared state. No NgRx or other state library.
- Native control flow (`@if`, `@for`, `@switch`) throughout; no `*ngIf`/`*ngFor`.
- App providers (`src/app/app.config.ts`): `provideBrowserGlobalErrorListeners()`, `provideZonelessChangeDetection()`, `provideRouter(routes, withComponentInputBinding())`, `provideHttpClient(withFetch(), withInterceptors([AUTH_INTERCEPTOR]))`, `provideAnimations()`, and `provideAppInitializer()` that awaits `firstValueFrom(AuthService.initAuth())`.
- Strict TypeScript (`strict`, `strictTemplates`, `noImplicitReturns`, `noPropertyAccessFromIndexSignature`, `noImplicitOverride`, `noFallthroughCasesInSwitch`) with path aliases `@app/*`, `@core/*`, `@shared/*`, `@features/*`.
- Tests: **Vitest 4** + jsdom 28 through the `@angular/build:unit-test` builder (its `runner` default is `vitest`; no Karma config exists). **63 `*.spec.ts` files** cover the app shell and routes, all of `core/`, every feature component/service/store, and the shared components/pipes/utils. The largest untested file is `shared/components/header/header.component.ts` (~757 lines, no spec).
- Formatting: Prettier 3 (`printWidth: 100`, single quotes, Angular parser for HTML); `.editorconfig` uses two-space indentation and single quotes.
- Linting: ESLint 10 with angular-eslint 21 (`eslint.config.js` flat config: `eslint.configs.recommended`, `tseslint.configs.recommended`/`stylistic`, `angular.configs.tsRecommended`, `templateRecommended`, `templateAccessibility`, plus `directive-selector` and `component-selector` rules requiring the `app` prefix); wired through the `lint` target in `angular.json`.
- No Dockerfile, SSR, or e2e runner (no Playwright/Cypress/Karma config; `frontend/src/tests/` is empty). `frontend/README.md` still mentions an `ng e2e` target that does not exist, and `.vscode/launch.json` points at the removed Karma debug URL — both are stale. The SPA has its own CI gate in `.github/workflows/frontend.yml`. `npm run build` produces `dist/` with the production configuration by default.

## Layout

```text
frontend/src/
├── main.ts, index.html, styles.scss
├── environments/{environment.ts, environment.development.ts}
└── app/
    ├── app.ts              # shell: skip link, header, router outlet, footer
    ├── app.config.ts       # providers and auth initializer
    ├── app.routes.ts       # lazy routes
    ├── core/
    │   ├── api/            ApiClient, ApiError/envelope, API_BASE_URL
    │   ├── auth/           AuthStore, AuthService, RefreshService, guards, interceptor
    │   ├── config/         APP_SETTINGS
    │   ├── idempotency/    IdempotencyKeyService
    │   └── toast/          ToastService
    ├── shared/
    │   ├── components/     header, footer, buttons, dialogs, form fields, pagination,
    │   │                   skeletons, status badges, toast host, product image,
    │   │                   review-form, review-list, ...
    │   ├── models/         auth, catalog/category, cart, customer/address, avatar,
    │   │                   order, payment, notification, wishlist, shipping
    │   ├── pipes/          money.pipe.ts
    │   └── utils/          datetime.ts, notification-text.ts
    └── features/
        ├── auth/           login, register
        ├── catalog/        home, product-card, product-detail,
        │                   ShowcaseApiService, CategoriesApiService
        ├── cart/           cart component, CartService, CartStore
        ├── wishlist/       wishlist component, WishlistService, WishlistStore, WishlistButton
        ├── orders/         order-list, order-details, OrdersApiService
        ├── payments/       payment-status, PaymentsApiService
        ├── shipping/       ShippingApiService (service only — no component)
        ├── customer/       profile, CustomerApiService, AvatarApiService
        └── notifications/  notifications component, NotificationsApiService
```

The app shell (`app.ts`) renders a skip link, `app-header`, a focusable `<main>`, `router-outlet`, and `app-footer`. The header owns public category navigation, auth links, cart count, account menu, avatar display, and the toast host.

`features/shipping/` contains **only** `shipping-api.service.ts` — there is no shipping UI. Ten of its sixteen methods (method/zone/rate admin CRUD, shipment process/dispatch/deliver, tracking-event creation, and the quote/shipment detail reads) have **no caller** outside the spec file; only `getMethods`, `createQuote`, `getShipments`, and `getTrackingEvents` are used by components.

## Routing

All routes are lazy `loadComponent` routes (`src/app/app.routes.ts`):

| Path | Component | Guard / purpose |
| --- | --- | --- |
| `/` | catalog home | public |
| `/products/:id` | product detail | public |
| `/login` | login | `guestGuard` |
| `/register` | register | `guestGuard` |
| `/cart` | cart | `authGuard` |
| `/wishlist` | wishlist | `authGuard` (UI route); API uses cookie auth |
| `/orders` | order list | `authGuard` |
| `/orders/:id` | order details | `authGuard` |
| `/payments/:id` | payment status | `authGuard` |
| `/profile` | customer profile | `authGuard` |
| `/notifications` | notifications | `authGuard` |
| `/home`, `/products` | catalog home | public aliases |
| `/categories`, `/brands` | catalog home | public UI placeholders |
| `/allorders` | redirect `/orders` | `pathMatch: 'full'` |
| `/settings` | customer profile | `authGuard` alias |
| `/cash-order` | cart | `authGuard` alias |
| `**` | redirect `/` | — |

`authGuard` sends unauthenticated users to `/login?returnUrl=<url>`; `guestGuard` sends authenticated users to `/`. The placeholder aliases exist in the route table but some reuse components that expect route parameters; do not treat them as complete feature surfaces.

## Bootstrap & Authentication

1. `main.ts` calls `bootstrapApplication(App, appConfig)`.
2. `provideAppInitializer()` invokes `AuthService.initAuth()` and waits for the `GET /v1/auth/me` request via `firstValueFrom`.
3. `initAuth()` treats an unauthenticated response as a normal state (`catchError` → complete), so bootstrap is never blocked by a missing session.
4. Protected routes rely on the resulting `AuthStore.isAuthenticated()` value; no token rehydration step exists.

- **Cookies, not storage**: the server sets encrypted `HttpOnly` `mm_at`/`mm_rt` cookies. `ApiClient` sends every request with `withCredentials: true`. There is no token storage service, `Authorization` header, or JavaScript-accessible token.
- **`AuthStore`** exposes read-only signals for `user`, `loading`, `error`, `isRefreshing`, and `accessTokenExpiresAt`, plus computed `isAuthenticated` and `userId`. `setSessionExpiry()` prefers the server's `expiresIn` over the absolute timestamp; `clear()` resets user/error/loading/refreshing/expiry.
- **`AuthService`** wraps `/v1/auth/login`, `/v1/auth/signup`, `/v1/auth/me`, `/v1/auth/logout`, and `/v1/auth/change-password`. Login, `/me`, and refresh store the user (where applicable) and session expiry. An Angular `effect` watches `accessTokenExpiresAt` and schedules a proactive refresh one minute before expiry, with retry/backoff for transient failures and a `visibilitychange` check for background tabs.
- **`RefreshService`** posts to `/v1/auth/refresh` with `skipAuth`/`skipRefresh` and deduplicates concurrent refreshes through a shared, completing observable. It updates expiry on success and clears the store only on `401`/`403`; transient errors leave the session intact.
- **`AUTH_INTERCEPTOR`** adds no header. On a `401`, unless `SKIP_REFRESH` or `REFRESH_ATTEMPTED` is set, it calls `RefreshService`, retries the original request once with cookies, and preserves the original error if refresh fails. The refresh service owns session clearing.
- **Logout** calls public `POST /v1/auth/logout` with `skipAuth`, clears the proactive-refresh timer and local store in `finalize`, and therefore works even when the access token has expired or the request fails.

## API Layer

- **`ApiClient`** (`core/api/api-client.ts`) is the single HTTP entry point. `API_BASE_URL` comes from `environment.apiBaseUrl` (via the `API_BASE_URL` injection token in `core/api/api-base-url.token.ts`); both environment files currently set it to `''`, meaning same-origin.
- It supports `get`, `post`, `patch`, `put`, `putFormData`, and `delete`; sets `withCredentials`; requests `observe: 'response'`; returns `204` as `undefined`; and unwraps the shared `{time, success, data, error, meta}` envelope. A `success: false` body throws `ApiError` even on HTTP 200.
- Request bodies are converted **camelCase → snake_case** for JSON mutations; response data is converted **snake_case → camelCase**. `FormData` is passed through unchanged.
- Paginated responses (`meta` present) are returned as `{data, meta}` with normalized `page`, `pageSize`, `total`, and `totalPages`.
- Failures become `ApiError` (`status`, `code`, `message`, optional `fields`, `originalError`) through `createApiError`; `HttpContextToken`s `SKIP_AUTH` and `SKIP_REFRESH` opt out of the auth interceptor.
- `ApiClientOptions` supports `skipAuth`, `skipRefresh`, and custom `headers` (used for `Idempotency-Key`). Feature services build query strings themselves.
- The used dev proxy is `frontend/proxy.conf.js`: `/v1` and `/swagger` → `http://localhost:8080`, with cookies rewritten to localhost. `proxy.conf.json` is present but unreferenced by `angular.json`. There are no `fileReplacements`, so `environment.ts` is the file actually imported (hard-coded in `api-base-url.token.ts`); `environment.development.ts` is currently unused.
- Feature API services:
  - `ShowcaseApiService` — public product list (keyset pagination via `page_size`/`cursor`/`category_id`, load-more) and product detail, plus paginated reviews, and `createReview` (posts to `/v1/products/{id}/reviews`, which requires the `reviews:write` permission server-side; the customer is taken from the session, so no customer id is sent).
  - `CategoriesApiService` — public category list.
  - `CartService` — authenticated cart tree; `checkout(idempotencyKey, shippingQuoteId, addressId)` posts `{shippingQuoteId, addressId}` with the `Idempotency-Key` header.
  - `CustomerApiService` — customer lookup/update and address CRUD/default.
  - `AvatarApiService` — avatar metadata/presigned URL and multipart upload.
  - `OrdersApiService` — paginated order list (`user_id`, `page`, `page_size`), order detail, cancel.
  - `PaymentsApiService` — payment create/read.
  - `ShippingApiService` — methods list, quote create/get, shipments by `order_id`/id, shipment tracking-history read, and admin method/zone/rate CRUD plus shipment process/dispatch/deliver/tracking-event creation.
  - `NotificationsApiService` — list/detail (scoped to caller), mark read/unread, mark-all-read, delete, and preferences read/replace.
  - `AuthService` — auth flows above.

## State & UX

- State is signal-based and local unless shared. Shared feature stores are `CartStore` (`cart`, `loading`, `error`; computed `itemCount`, `estimatedTotal`, `isEmpty`; clears when auth is lost) and `WishlistStore`; cart mutations from any feature immediately update `CartStore`, so the header badge always reflects the current `itemCount`.
- `ToastService` holds a signal array of auto-dismissing toasts; `ToastHostComponent` renders them. Feature components commonly hold their own `loading`, `error`, `saving`, and dialog signals.
- `APP_SETTINGS` (`core/config/app-settings.ts`) centralizes `currency: 'BRL'`, `paymentMethod: 'card'`, default page size 20, max page size 100, and default page 1.
- `money.pipe.ts` formats integer cents; status/type labels live beside their models.
- The UI uses skeleton loaders, error states with retry, empty states, confirmation dialogs, status badges, and accessibility attributes (skip link, ARIA labels, focus handling).
- **`IdempotencyKeyService`** generates `crypto.randomUUID()` keys in `sessionStorage` under `mm_checkout_{userId}_{cartId}` and `mm_payment_{orderId}`, with matching clear helpers. It also exposes `newQuoteCycleKey()` (not persisted): the cart derives one `Idempotency-Key` per shipping method from a per-round cycle key (`<cycle>:<methodId>`), so retrying a round replays the same quotes. Order details separately stores the created payment id at `mm_payment_{orderId}`.

## Feature Behavior

- **Catalog home** reads `page` and `category_id` query params, loads public showcase products and categories, and paginates. Product cards show real stock, rating (`total_rating / total_reviews`), and the primary presigned image. Add-to-cart uses `CartService` and updates the shared `CartStore` when authenticated, redirecting to login otherwise.
- **Product detail** loads the public showcase view plus its approved reviews in parallel, renders real gallery images (with monogram fallback), stock/SKU, description, specifications, and a reviews tab, and disables add-to-cart when out of stock. Authenticated users can open a review form (`ReviewFormComponent`) and submit `{description, rating}` through `ShowcaseApiService.createReview`; the API requires the `reviews:write` permission, which the `customer` role holds. The displayed average rating is computed **client-side** from the loaded review page by `ReviewListComponent` — the backend has no review-summary HTTP endpoint. Add-to-cart updates the shared `CartStore`; the wishlist button toggles the shared `WishlistStore`.
- **Wishlist** displays the authenticated user's wishlist with product cards; supports add/remove/clear via `WishlistService`, which updates the shared `WishlistStore`. The `WishlistButtonComponent` toggles the heart icon on product cards and in the catalog. Wishlist endpoints are authenticated (`wishlist:read`/`wishlist:write`); `WishlistService` relies on the HttpOnly cookie like the cart and sends no `user_id`.
- **Cart** supports add/update/remove/clear and checkout with **real shipping**: it loads the customer's addresses (auto-selecting the default), loads active shipping methods, and requests a quote per method through `ShippingApiService.createQuote(methodId, addressId, subtotalCents, idempotencyKey)`, keeping them in a `Map<methodId, ShippingQuote>`. A round of quotes derives one key per method from `IdempotencyKeyService.newQuoteCycleKey()`; the round is re-run when the subtotal/address/method set changes. Checkout requires an address and an `active` quote, then calls `CartService.checkout(idempotencyKey, quote.id, addressId)` with the `Idempotency-Key` header; on success it clears the idempotency key/store and navigates to `/orders/:orderId`. The displayed total adds the quote's shipping cost, but the backend order total remains authoritative.
- **Orders** — `/orders` lists the user's paginated orders (cards with status badge, item summary, total, link to details). `/orders/:id` loads an order by route id, supports cancellation while the status is cancellable, can create a payment for an unpaid order, and polls the payment status. The details template renders a shipping section (method, destination, tracking code, carrier, shipped/delivered timestamps, tracking events) from `order.shippingCostCents`, `order.shippingMethodName`, `order.destinationSnapshot`, plus the separately loaded `shipment`/`trackingEvents` signals. The order response carries the persisted shipping snapshot; the shipment itself is fetched separately through `ShippingApiService.getShipments(orderId)` and its tracking history through `ShippingApiService.getTrackingEvents(shipmentId)`, then kept in the component's own `shipment`/`trackingEvents` signals (the order API never returns a shipment).
- **Payment status** polls `/v1/payments/{id}` every 5 seconds while `pending` and compares `userId` to the current session before showing it.
- **Profile** has account, security, addresses, and notification-preference sections; it loads the customer by `user_id`, the authenticated account, avatar, addresses, and preferences, and uses reactive forms with version fields for customer/address updates.
- **Notifications** lists the signed-in user's notifications and filters client-side. Read/unread, mark-all-read, and delete call the real `/v1/notifications` endpoints, and the component updates its local signal on success.

## Commands

```bash
cd frontend
npm start        # ng serve; proxy /v1 and /swagger to localhost:8080
npm run build    # production build into dist/
npm test         # Vitest via ng test
npm run lint     # ESLint via angular-eslint
npm run format   # Prettier --write .
npm run format:check  # Prettier --check .
```

## Conventions

- File suffixes: `*.component.ts`, `*.service.ts`, `*.store.ts`, `*.guard.ts`, `*.interceptor.ts`, `*.models.ts`, `*.pipe.ts`, `*.spec.ts`.
- Components are standalone, use `inject()` rather than constructor injection, and keep state in signals.
- Components call feature API services; only those services (or `core`) use `ApiClient`. Components never inject `HttpClient`.
- Cross-folder imports use the path aliases; imports within a feature may be relative.
- Treat the wire format as snake_case and the TypeScript model layer as camelCase; let `ApiClient` translate at the boundary.
- Styling is SCSS with CSS custom properties (`var(--color-*)`); global design tokens live in `src/styles.scss`.
- The SPA is a client of the API only. It must not define authoritative prices, totals, stock, ownership, or order/payment transitions; those belong to the backend.
- Auth is cookie-only. Never add token persistence or an `Authorization` header.

## Known Frontend / API Gaps

- **Notifications read state is persisted.** `read_at` is stored server-side; mark-read, mark-unread, mark-all-read, and delete call `/v1/notifications` endpoints.
- **Order shipping UI is partial.** `GET /v1/orders/{id}` returns the persisted shipping snapshot (quote, method, method name, cost, destination) but never a shipment. The shipment and its tracking history are separate shipping reads (`GET /v1/shipping/shipments?order_id=` and `GET /v1/shipping/shipments/{id}/events`), so the order-details component keeps them in their own signals instead of merging them into the order. The shipment is read again while the order is paid, because the shipping domain creates it asynchronously when the order is paid. An order placed without shipping shows no shipping section.
- **Review submission needs the `reviews:write` permission.** The review form posts to `/v1/products/{id}/reviews`, which maps to `reviews:write`; the `customer` role holds it, so a signed-in customer can submit. Reviews are only visible after async moderation approves them.
- **Shipping methods require authentication.** `GET /v1/shipping/methods` is not public, so an anonymous visitor cannot see delivery options; the cart loads them only for signed-in users.
- **Quote pricing follows the cart subtotal.** The cart posts its `estimatedTotalCents` as `subtotal_cents` when it quotes a method, and re-quotes when the cart total or the delivery address changes; the quote response carries `free_threshold_cents`, which drives the free-shipping progress bar. The amount charged is still decided by the server: the checkout re-prices the quote with the subtotal the cart computed.
- **Placeholder routes are incomplete.** `/allorders` redirects to `/orders`; `/home`, `/products`, `/categories`, `/brands`, `/settings`, and `/cash-order` are aliases that reuse other components.
- **Dev/build config has drift.** `proxy.conf.json` is stale and `environment.development.ts` is not selected by any build configuration.
- **Request bodies are not uniformly camelCase.** `ShippingApiService.createQuote`/`dispatchShipment`/`addTrackingEvent` and `AuthService.changePassword` already send snake_case keys; `ApiClient.camelToSnake` is a no-op on them, so it works, but it diverges from the rest of the codebase. `ShippingApiService` also keeps its `rxjs/operators` and model imports at the bottom of the file.
- **No shipping admin UI.** `ShippingApiService` exposes ten endpoints (method/zone/rate CRUD, shipment lifecycle transitions, tracking-event creation, quote/shipment detail) that no component calls.
- **No review-summary endpoint.** The backend exposes `GetReviewSummary`/`GetRatingSummary` as application use cases only; the SPA derives the average from the loaded reviews and the showcase product's `total_rating`/`total_reviews` counters.
- **`POST /v1/orders` is never called by the SPA.** Orders are created through `POST /v1/cart/checkout`, so the client sends `Idempotency-Key` on exactly three of the four server-required paths (`/v1/cart/checkout`, `/v1/payments`, `/v1/shipping/quotes`).