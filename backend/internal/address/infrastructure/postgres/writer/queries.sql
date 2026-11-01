-- name: CreateAddress :execrows
INSERT INTO addresses (id, customer_id, is_default, label, full_name, street, city, state, zip, country, phone, created_at, updated_at, version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, 1)
ON CONFLICT (id) DO NOTHING;

-- name: UpdateAddress :one
UPDATE addresses
SET label = $2,
    full_name = $3,
    street = $4,
    city = $5,
    state = $6,
    zip = $7,
    country = $8,
    phone = $9,
    updated_at = NOW(),
    version = version + 1
WHERE id = $1
  AND version = $10
RETURNING id, customer_id, is_default, label, full_name, street, city, state, zip, country, phone, created_at, updated_at, version;

-- name: DeleteAddress :execrows
DELETE FROM addresses
WHERE id = $1 AND customer_id = $2;

-- name: UnsetOldDefault :execrows
UPDATE addresses
SET is_default = FALSE,
    updated_at = NOW(),
    version = version + 1
WHERE customer_id = $1 AND is_default = TRUE;

-- name: SetNewDefault :one
UPDATE addresses
SET is_default = TRUE,
    updated_at = NOW(),
    version = version + 1
WHERE id = $1 AND customer_id = $2
RETURNING id, customer_id, is_default, label, full_name, street, city, state, zip, country, phone, created_at, updated_at, version;

-- name: AddressExists :one
SELECT EXISTS (
    SELECT 1
    FROM addresses
    WHERE id = $1
);