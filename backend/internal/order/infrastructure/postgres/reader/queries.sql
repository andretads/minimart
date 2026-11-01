-- name: GetOrder :one
SELECT id, user_id, total_cents, created_at, status, cancelled_at, idempotency_key, customer_id,
       shipping_quote_id, shipping_method_id, shipping_method_name, shipping_cost_cents, destination_snapshot
FROM orders
WHERE id = $1;

-- name: GetOrderByIdempotencyKey :one
SELECT id, user_id, total_cents, created_at, status, cancelled_at, idempotency_key, customer_id,
       shipping_quote_id, shipping_method_id, shipping_method_name, shipping_cost_cents, destination_snapshot
FROM orders
WHERE idempotency_key = $1;

-- name: ListOrderItems :many
SELECT product_id, name, unit_price_cents, quantity, line_total_cents
FROM order_items
WHERE order_id = $1
ORDER BY id;

-- name: ListOrdersByUserID :many
SELECT id, user_id, total_cents, created_at, status, cancelled_at, idempotency_key, customer_id,
       shipping_quote_id, shipping_method_id, shipping_method_name, shipping_cost_cents, destination_snapshot
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: CountOrdersByUserID :one
SELECT COUNT(*) FROM orders WHERE user_id = $1;

-- name: ListOrderItemsByOrderIDs :many
SELECT order_id, product_id, name, unit_price_cents, quantity, line_total_cents
FROM order_items
WHERE order_id = ANY(sqlc.arg(order_ids)::text[])
ORDER BY order_id, id;

-- name: ListPlacedOrdersBefore :many
SELECT id, user_id, total_cents, created_at, status, cancelled_at, idempotency_key, customer_id,
       shipping_quote_id, shipping_method_id, shipping_method_name, shipping_cost_cents, destination_snapshot
FROM orders
WHERE status = 'placed' AND created_at <= $1
ORDER BY created_at, id
LIMIT $2;
