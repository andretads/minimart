-- name: CreateAccount :execrows
INSERT INTO accounts (id, username, email, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $5)
ON CONFLICT (username) DO NOTHING;

-- name: UpdatePassword :execrows
UPDATE accounts
SET password_hash = $2, updated_at = $3
WHERE id = $1;

-- name: AssignRole :exec
INSERT INTO user_roles (account_id, role_id)
VALUES ($1, $2)
ON CONFLICT (account_id, role_id) DO NOTHING;

-- name: CreateRole :exec
INSERT INTO roles (id, description, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO NOTHING;

-- name: CreatePermission :exec
INSERT INTO permissions (id, created_at)
VALUES ($1, $2)
ON CONFLICT (id) DO NOTHING;

-- name: AddRolePermission :exec
INSERT INTO role_permission (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT (role_id, permission_id) DO NOTHING;
