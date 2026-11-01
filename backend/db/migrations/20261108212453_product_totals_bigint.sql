-- migrate:up

ALTER TABLE products
    ALTER COLUMN total_rating TYPE BIGINT,
    ALTER COLUMN total_reviews TYPE BIGINT;

-- migrate:down

ALTER TABLE products
    ALTER COLUMN total_rating TYPE INT,
    ALTER COLUMN total_reviews TYPE INT;
