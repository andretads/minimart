-- name: SaveProduct :exec
INSERT INTO products (id, name, price_cents, stock, version, created_at, updated_at, category_id, sku, description, details, total_rating, total_reviews)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO UPDATE
SET name              = EXCLUDED.name,
    price_cents       = EXCLUDED.price_cents,
    stock             = EXCLUDED.stock,
    category_id       = EXCLUDED.category_id,
    sku               = EXCLUDED.sku,
    description       = EXCLUDED.description,
    details           = EXCLUDED.details,
    total_rating      = EXCLUDED.total_rating,
    total_reviews     = EXCLUDED.total_reviews,
    updated_at        = NOW();

-- name: UpdateProduct :one
UPDATE products
SET name              = $2,
    price_cents       = $3,
    category_id       = $4,
    sku               = $5,
    description       = $6,
    details           = $7,
    updated_at        = NOW(),
    version           = version + 1
WHERE id = $1
  AND version = $8
RETURNING id, version, updated_at;

-- name: ProductExists :one
SELECT EXISTS (
    SELECT 1
    FROM products
    WHERE id = $1
);

-- name: CreateStockReservation :execrows
INSERT INTO stock_reservations (order_id, product_id, quantity)
VALUES ($1, $2, $3)
ON CONFLICT (order_id, product_id) DO NOTHING;

-- ReserveProductStock decrements stock only while enough is available. The
-- "stock >= quantity" predicate is the atomic guard against overselling under
-- concurrency: 0 rows means the product is sold out (or missing), never a lost
-- update. version is bumped so a concurrent name/price edit is invalidated.
-- name: ReserveProductStock :execrows
UPDATE products
SET stock      = stock - sqlc.arg(quantity),
    version    = version + 1,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND stock >= sqlc.arg(quantity);

-- ReleaseStockReservations marks every outstanding reservation for an order as
-- released and returns them, so the caller can restore each product's stock. A
-- second call returns no rows: release is idempotent.
-- name: ReleaseStockReservations :many
UPDATE stock_reservations
SET released_at = NOW()
WHERE order_id = sqlc.arg(order_id)
  AND released_at IS NULL
RETURNING product_id, quantity;

-- name: RestoreProductStock :exec
UPDATE products
SET stock      = stock + sqlc.arg(quantity),
    version    = version + 1,
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: SaveReview :exec
INSERT INTO product_reviews (id, product_id, customer_id, description, rating, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateReviewStats :exec
UPDATE products
SET total_rating  = $2,
    total_reviews = $3,
    updated_at    = NOW(),
    version       = version + 1
WHERE id = $1;

-- name: UpsertRatingSummary :exec
INSERT INTO product_rating_summary (product_id, total_reviews, rating_sum, average_rating, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (product_id) DO UPDATE
SET total_reviews = EXCLUDED.total_reviews,
    rating_sum = EXCLUDED.rating_sum,
    average_rating = EXCLUDED.average_rating,
    updated_at = NOW();

-- BumpProductVersion also locks the product row, serializing image writes for
-- the same product so position assignment stays deterministic.
-- name: BumpProductVersion :execrows
UPDATE products
SET version    = version + 1,
    updated_at = NOW()
WHERE id = $1;

-- name: NextProductImagePosition :one
SELECT COALESCE(MAX(position) + 1, 0)::int
FROM product_images
WHERE product_id = $1;

-- name: SaveProductImage :one
INSERT INTO product_images (id, product_id, object_key, bucket, content_type, size_bytes, position, version, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, 1, $8, $9)
RETURNING id, product_id, object_key, bucket, content_type, size_bytes, position, version, created_at, updated_at;

-- name: DeleteProductImage :one
DELETE FROM product_images
WHERE id = $1
  AND product_id = $2
RETURNING id, product_id, object_key, bucket, content_type, size_bytes, position, version, created_at, updated_at;

-- name: UpdateReviewModeration :exec
UPDATE product_reviews
SET status = $2,
    moderated_at = $3,
    moderation_decision = $4,
    moderation_confidence = $5,
    moderation_categories = $6,
    moderation_explanation = $7,
    moderation_model = $8,
    moderation_model_version = $9
WHERE id = $1;
