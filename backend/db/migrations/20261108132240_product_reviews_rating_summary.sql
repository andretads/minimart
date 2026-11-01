-- migrate:up

CREATE UNIQUE INDEX IF NOT EXISTS uq_product_reviews_product_customer
    ON product_reviews (product_id, customer_id);

CREATE INDEX IF NOT EXISTS idx_product_reviews_product_status_created
    ON product_reviews (product_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS product_rating_summary (
    product_id TEXT PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
    total_reviews BIGINT NOT NULL DEFAULT 0,
    rating_sum BIGINT NOT NULL DEFAULT 0,
    average_rating DECIMAL(10, 4) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- migrate:down

DROP TABLE IF EXISTS product_rating_summary;
DROP INDEX IF EXISTS idx_product_reviews_product_status_created;
DROP INDEX IF EXISTS uq_product_reviews_product_customer;
