-- name: CreatePayment :execrows
INSERT INTO payments (id, order_id, user_id, idempotency_key, amount_cents, currency, method, status, provider)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (idempotency_key) DO NOTHING;

-- name: UpdatePaymentStatus :execrows
UPDATE payments
SET status = $2, provider_ref = $3, failure_reason = $4, updated_at = $5
WHERE id = $1 AND status = 'pending';

-- name: ReconcilePayment :execrows
UPDATE payments
SET status = $2, provider_ref = $3, failure_reason = $4, updated_at = $5
WHERE id = $1 AND status IN ('pending', 'unknown');

-- name: RefundPayment :execrows
UPDATE payments
SET status = $2, provider_ref = $3, failure_reason = $4, updated_at = $5
WHERE id = $1 AND status IN ('pending', 'succeeded', 'unknown');
