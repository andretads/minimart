-- name: CreateCustomer :execrows
INSERT INTO customers (id, user_id, display_name, status, created_at, updated_at)
VALUES ($1, $2, $3, 'active', $4, $4)
ON CONFLICT (user_id) DO NOTHING;

-- name: UpdateCustomerProfile :one
UPDATE customers
SET display_name = $2,
    updated_at = NOW(),
    version = version + 1
WHERE id = $1
  AND version = $3
RETURNING id, version, updated_at;

-- name: DeactivateCustomer :one
UPDATE customers
SET status = 'deactivated',
    deactivated_at = COALESCE(deactivated_at, NOW()),
    updated_at = NOW(),
    version = version + 1
WHERE id = $1
RETURNING id, version, updated_at, deactivated_at;

-- name: CustomerExists :one
SELECT EXISTS (
    SELECT 1
    FROM customers
    WHERE id = $1
);