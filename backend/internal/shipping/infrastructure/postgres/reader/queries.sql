-- name: GetMethod :one
SELECT id, name, description, active, estimated_min_days, estimated_max_days, created_at, updated_at
FROM shipping_methods
WHERE id = $1;

-- name: ListActiveMethods :many
SELECT id, name, description, active, estimated_min_days, estimated_max_days, created_at, updated_at
FROM shipping_methods
WHERE active = TRUE
ORDER BY name;

-- name: ListAllMethods :many
SELECT id, name, description, active, estimated_min_days, estimated_max_days, created_at, updated_at
FROM shipping_methods
ORDER BY name;

-- name: GetZone :one
SELECT id, name, country, state, city, postal_code_start, postal_code_end, created_at, updated_at
FROM shipping_zones
WHERE id = $1;

-- name: ListZones :many
SELECT id, name, country, state, city, postal_code_start, postal_code_end, created_at, updated_at
FROM shipping_zones
ORDER BY name;

-- name: GetRate :one
SELECT id, zone_id, method_id, base_cost_cents, free_threshold_cents, active, created_at, updated_at
FROM shipping_rates
WHERE id = $1;

-- name: ListRates :many
SELECT id, zone_id, method_id, base_cost_cents, free_threshold_cents, active, created_at, updated_at
FROM shipping_rates
WHERE (sqlc.arg(zone_id)::text = '' OR zone_id = sqlc.arg(zone_id)::text) AND (sqlc.arg(method_id)::text = '' OR method_id = sqlc.arg(method_id)::text)
ORDER BY zone_id, method_id;

-- name: ListRatesByZone :many
SELECT id, zone_id, method_id, base_cost_cents, free_threshold_cents, active, created_at, updated_at
FROM shipping_rates
WHERE zone_id = $1
ORDER BY method_id;

-- name: ListRatesByMethod :many
SELECT id, zone_id, method_id, base_cost_cents, free_threshold_cents, active, created_at, updated_at
FROM shipping_rates
WHERE method_id = $1
ORDER BY zone_id;

-- name: GetQuote :one
SELECT * FROM shipping_quotes
WHERE id = $1;

-- name: GetQuoteByIdempotencyKey :one
SELECT * FROM shipping_quotes
WHERE idempotency_key = $1 AND idempotency_key <> '';

-- name: GetShipment :one
SELECT id, order_id, user_id, method_id, tracking_code, carrier, shipping_cost_cents, destination_snapshot, status, shipped_at, delivered_at, created_at, updated_at
FROM shipments
WHERE id = $1;

-- name: GetShipmentByOrderID :one
SELECT id, order_id, user_id, method_id, tracking_code, carrier, shipping_cost_cents, destination_snapshot, status, shipped_at, delivered_at, created_at, updated_at
FROM shipments
WHERE order_id = $1;

-- name: ListShipments :many
SELECT id, order_id, user_id, method_id, tracking_code, carrier, shipping_cost_cents, destination_snapshot, status, shipped_at, delivered_at, created_at, updated_at
FROM shipments
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountShipments :one
SELECT COUNT(*) FROM shipments WHERE user_id = $1;

-- name: GetTrackingEvents :many
SELECT id, shipment_id, status, description, location, occurred_at, created_at
FROM shipment_tracking_events
WHERE shipment_id = $1
ORDER BY occurred_at;