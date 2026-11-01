-- name: GetCategory :one
SELECT id, name, icon, version, created_at, updated_at
FROM categories
WHERE id = $1;

-- name: ListCategories :many
SELECT id, name, icon, version, created_at, updated_at
FROM categories
ORDER BY LOWER(name), id;