-- name: GetAddressByID :one
SELECT id, customer_id, is_default, label, full_name, street, city, state, zip, country, phone, created_at, updated_at, version
FROM addresses
WHERE id = $1;

-- name: GetAddressesByCustomerID :many
SELECT id, customer_id, is_default, label, full_name, street, city, state, zip, country, phone, created_at, updated_at, version
FROM addresses
WHERE customer_id = $1
ORDER BY is_default DESC, created_at DESC;

-- name: GetDefaultAddressByCustomerID :one
SELECT id, customer_id, is_default, label, full_name, street, city, state, zip, country, phone, created_at, updated_at, version
FROM addresses
WHERE customer_id = $1 AND is_default = TRUE;