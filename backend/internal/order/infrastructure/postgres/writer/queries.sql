-- name: CreateOrder :execrows
INSERT INTO orders (
    id, user_id, customer_id, total_cents, idempotency_key,
    shipping_quote_id, shipping_method_id, shipping_method_name, shipping_cost_cents, destination_snapshot
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (idempotency_key) WHERE idempotency_key <> '' DO NOTHING;

-- name: CreateOrderItem :exec
INSERT INTO order_items (order_id, product_id, name, unit_price_cents, quantity, line_total_cents)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateOrderStatus :execrows
UPDATE orders
SET status = sqlc.arg(new_status), cancelled_at = sqlc.arg(cancelled_at)
WHERE id = sqlc.arg(id) AND status = sqlc.arg(expected_status);

-- name: OrderExists :one
SELECT EXISTS (SELECT 1 FROM orders WHERE id = sqlc.arg(id));
