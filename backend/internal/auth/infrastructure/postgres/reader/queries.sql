-- name: GetAccountByID :one
SELECT id, username, email, password_hash, created_at, updated_at
FROM accounts
WHERE id = $1;

-- name: GetAccountByUsername :one
SELECT id, username, email, password_hash, created_at, updated_at
FROM accounts
WHERE username = $1;

-- name: GetRole :one
SELECT id, description, created_at FROM roles WHERE id = $1;

-- name: GetPermission :one
SELECT id, created_at FROM permissions WHERE id = $1;

-- name: GetRolesForAccount :many
SELECT r.id, r.description, r.created_at
FROM roles r
JOIN user_roles ur ON r.id = ur.role_id
WHERE ur.account_id = $1;

-- name: GetPermissionsForAccount :many
SELECT p.id
FROM permissions p
JOIN role_permission rp ON p.id = rp.permission_id
JOIN user_roles ur ON rp.role_id = ur.role_id
WHERE ur.account_id = $1;

-- name: GetAccountsByUsername :many
SELECT id FROM accounts WHERE username = ANY($1::text[]);
