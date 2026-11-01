-- name: CreateCategory :execrows
INSERT INTO categories (id, name, icon, version, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $5)
ON CONFLICT (LOWER(name)) DO NOTHING;

-- name: UpdateCategory :one
UPDATE categories
SET name       = $2,
    icon       = $3,
    updated_at = NOW(),
    version    = version + 1
WHERE id = $1
  AND version = $4
RETURNING id, version, updated_at;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE id = $1
  AND version = $2;

-- name: CategoryExists :one
SELECT EXISTS (
    SELECT 1
    FROM categories
    WHERE id = $1
);