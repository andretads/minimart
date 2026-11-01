-- name: GetCustomerByID :one
SELECT id, user_id, display_name, status, created_at, updated_at, deactivated_at, version
FROM customers
WHERE id = $1;

-- name: GetCustomerByUserID :one
SELECT id, user_id, display_name, status, created_at, updated_at, deactivated_at, version
FROM customers
WHERE user_id = $1;