-- name: GetProduct :one
SELECT id, name, price_cents, stock, version, created_at, updated_at, category_id,
       sku, description, details, total_rating, total_reviews
FROM products
WHERE id = $1;

-- name: ListProducts :many
SELECT id, name, price_cents, stock, version, created_at, updated_at, category_id,
       sku, description, details, total_rating, total_reviews
FROM products
ORDER BY created_at;

-- name: ListProductsByCategory :many
SELECT id, name, price_cents, stock, version, created_at, updated_at, category_id,
       sku, description, details, total_rating, total_reviews
FROM products
WHERE category_id = $1
ORDER BY created_at;

-- name: GetReviews :many
SELECT id, product_id, customer_id, description, rating, status, created_at
FROM product_reviews
WHERE product_id = $1 AND status = 'approved'
ORDER BY created_at DESC;

-- name: GetReviewsPaginated :many
SELECT id, product_id, customer_id, description, rating, status, created_at
FROM product_reviews
WHERE product_id = $1 AND status = 'approved'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountReviews :one
SELECT COUNT(*)::bigint
FROM product_reviews
WHERE product_id = $1 AND status = 'approved';

-- name: GetRatingSummary :one
SELECT product_id, total_reviews, rating_sum, average_rating, updated_at
FROM product_rating_summary
WHERE product_id = $1;

-- name: GetReviewByID :one
SELECT id, product_id, customer_id, description, rating, status, created_at,
       moderated_at, moderation_decision, moderation_confidence, moderation_categories,
       moderation_explanation, moderation_model, moderation_model_version
FROM product_reviews
WHERE id = $1;

-- name: ListProductImages :many
SELECT id, product_id, object_key, bucket, content_type, size_bytes, position, version, created_at, updated_at
FROM product_images
WHERE product_id = $1
ORDER BY position, created_at;

-- name: ListProductImagesByProductIDs :many
SELECT id, product_id, object_key, bucket, content_type, size_bytes, position, version, created_at, updated_at
FROM product_images
WHERE product_id = ANY(sqlc.arg(product_ids)::text[])
ORDER BY product_id, position, created_at;
