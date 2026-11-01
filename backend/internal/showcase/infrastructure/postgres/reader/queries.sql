-- name: ListProducts :many
SELECT *
FROM products
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: ListProductsAfter :many
SELECT *
FROM products
WHERE (created_at, id) < (sqlc.arg(created_at)::timestamptz, sqlc.arg(id)::text)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: ListProductsByCategory :many
SELECT *
FROM products
WHERE category_id = sqlc.arg(category_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: ListProductsByCategoryAfter :many
SELECT *
FROM products
WHERE category_id = sqlc.arg(category_id)
  AND (created_at, id) < (sqlc.arg(created_at)::timestamptz, sqlc.arg(id)::text)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: GetProduct :one
SELECT *
FROM products WHERE id = $1;

-- name: ListProductImages :many
SELECT id, product_id, object_key, content_type, size_bytes, position
FROM product_images
WHERE product_id = $1
ORDER BY position, created_at;

-- name: ListProductImagesByProductIDs :many
SELECT id, product_id, object_key, content_type, size_bytes, position
FROM product_images
WHERE product_id = ANY(sqlc.arg(product_ids)::text[])
ORDER BY product_id, position, created_at;

-- name: ListApprovedReviewsPaginated :many
SELECT id, product_id, customer_id, description, rating, status, created_at
FROM product_reviews
WHERE product_id = $1 AND status = 'approved'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountApprovedReviews :one
SELECT COUNT(*)::bigint
FROM product_reviews
WHERE product_id = $1 AND status = 'approved';

-- name: ListCategories :many
SELECT id, name, icon
FROM categories
ORDER BY LOWER(name), id;
