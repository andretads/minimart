-- name: CreateMethod :exec
INSERT INTO shipping_methods (id, name, description, active, estimated_min_days, estimated_max_days, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateMethod :exec
UPDATE shipping_methods
SET name = $2, description = $3, active = $4, estimated_min_days = $5, estimated_max_days = $6, updated_at = $7
WHERE id = $1;

-- name: CreateZone :exec
INSERT INTO shipping_zones (id, name, country, state, city, postal_code_start, postal_code_end, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: UpdateZone :exec
UPDATE shipping_zones
SET name = $2, country = $3, state = $4, city = $5, postal_code_start = $6, postal_code_end = $7, updated_at = $8
WHERE id = $1;

-- name: CreateRate :exec
INSERT INTO shipping_rates (id, zone_id, method_id, base_cost_cents, free_threshold_cents, active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateRate :exec
UPDATE shipping_rates
SET base_cost_cents = $2, free_threshold_cents = $3, active = $4, updated_at = $5
WHERE id = $1;

-- name: CreateQuote :execrows
INSERT INTO shipping_quotes (id, customer_id, method_id, address_id, idempotency_key, destination_snapshot, shipping_cost_cents, subtotal_cents, free_threshold_cents, estimated_min_days, estimated_max_days, status, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (idempotency_key) WHERE idempotency_key <> '' DO NOTHING;

-- name: UpdateQuote :exec
UPDATE shipping_quotes
SET status = $2, expires_at = $3, shipping_cost_cents = $4, subtotal_cents = $5, free_threshold_cents = $6
WHERE id = $1;

-- name: ConsumeQuote :execrows
UPDATE shipping_quotes
SET status = sqlc.arg(new_status)
WHERE id = $1 AND status = 'active';

-- name: ExpireQuotes :exec
UPDATE shipping_quotes
SET status = 'expired'
WHERE status = 'active' AND expires_at < NOW();

-- name: CreateShipment :exec
INSERT INTO shipments (id, order_id, user_id, method_id, tracking_code, carrier, shipping_cost_cents, destination_snapshot, status, shipped_at, delivered_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: UpdateShipment :exec
UPDATE shipments
SET tracking_code = $2, carrier = $3, shipping_cost_cents = $4, destination_snapshot = $5, status = $6, shipped_at = $7, delivered_at = $8, updated_at = $9
WHERE id = $1;

-- name: AddTrackingEvent :exec
INSERT INTO shipment_tracking_events (id, shipment_id, status, description, location, occurred_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);