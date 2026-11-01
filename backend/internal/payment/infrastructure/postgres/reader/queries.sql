-- name: GetPayment :one
SELECT id, order_id, idempotency_key, amount_cents, currency, method, status, provider, provider_ref, failure_reason, created_at, updated_at, user_id
FROM payments
WHERE id = $1;

-- name: GetPaymentByIdempotencyKey :one
SELECT id, order_id, idempotency_key, amount_cents, currency, method, status, provider, provider_ref, failure_reason, created_at, updated_at, user_id
FROM payments
WHERE idempotency_key = $1;

-- name: ListReconcilablePayments :many
SELECT id, order_id, idempotency_key, amount_cents, currency, method, status, provider, provider_ref, failure_reason, created_at, updated_at, user_id
FROM payments
WHERE status IN ('pending', 'unknown')
  AND updated_at < $1
ORDER BY updated_at ASC
LIMIT $2;

-- name: ListSucceededPaymentsByOrderID :many
SELECT id, order_id, idempotency_key, amount_cents, currency, method, status, provider, provider_ref, failure_reason, created_at, updated_at, user_id
FROM payments
WHERE order_id = $1 AND status = 'succeeded'
ORDER BY created_at ASC;
