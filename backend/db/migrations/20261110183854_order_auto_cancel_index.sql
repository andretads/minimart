-- migrate:up
CREATE INDEX IF NOT EXISTS orders_placed_created_at_idx ON orders (created_at, id) WHERE status = 'placed';

-- migrate:down
DROP INDEX IF EXISTS orders_placed_created_at_idx;
