-- migrate:up

CREATE INDEX IF NOT EXISTS products_created_at_id_idx
    ON products (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS products_category_created_at_id_idx
    ON products (category_id, created_at DESC, id DESC);

-- migrate:down

DROP INDEX IF EXISTS products_category_created_at_id_idx;
DROP INDEX IF EXISTS products_created_at_id_idx;
