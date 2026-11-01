-- migrate:up

CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE permissions (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    email TEXT
);

CREATE UNIQUE INDEX idx_accounts_email ON accounts (email) WHERE email IS NOT NULL;

CREATE TABLE user_roles (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (account_id, role_id)
);

CREATE TABLE role_permission (
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE customers (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    display_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    deactivated_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1,
    CONSTRAINT customers_status_check CHECK (status IN ('active', 'deactivated'))
);

CREATE TABLE addresses (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    label TEXT NOT NULL,
    full_name TEXT NOT NULL,
    street TEXT NOT NULL,
    city TEXT NOT NULL,
    state TEXT NOT NULL,
    zip TEXT NOT NULL,
    country TEXT NOT NULL,
    phone TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX addresses_customer_default_unique ON addresses (customer_id) WHERE is_default;

CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    icon TEXT NOT NULL DEFAULT '',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX categories_name_lower_idx ON categories (LOWER(name));

CREATE TABLE products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    price_cents BIGINT NOT NULL,
    stock INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1,
    category_id TEXT NOT NULL REFERENCES categories(id),
    description TEXT,
    sku TEXT UNIQUE,
    details JSONB DEFAULT '[]'::jsonb,
    total_rating INT DEFAULT 0,
    total_reviews INT DEFAULT 0
);

CREATE INDEX products_category_id_idx ON products (category_id);

CREATE TABLE product_images (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL,
    bucket TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    position INT NOT NULL DEFAULT 0 CHECK (position >= 0),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX product_images_product_position_idx ON product_images (product_id, position, created_at);

CREATE TABLE product_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    customer_id TEXT NOT NULL,
    description TEXT NOT NULL,
    rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'needs_review')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    moderated_at TIMESTAMPTZ,
    moderation_decision TEXT CHECK (moderation_decision IN ('approved', 'rejected', 'needs_review')),
    moderation_confidence DOUBLE PRECISION,
    moderation_categories JSONB DEFAULT '[]'::jsonb,
    moderation_explanation TEXT,
    moderation_model TEXT,
    moderation_model_version TEXT
);

CREATE INDEX idx_product_reviews_product ON product_reviews (product_id);
CREATE INDEX idx_product_reviews_customer ON product_reviews (customer_id);
CREATE INDEX idx_product_reviews_status ON product_reviews (status);
CREATE INDEX idx_product_reviews_moderation_status ON product_reviews (moderation_decision);

CREATE TABLE orders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    total_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'placed',
    cancelled_at TIMESTAMPTZ,
    idempotency_key TEXT NOT NULL DEFAULT '',
    customer_id TEXT REFERENCES customers(id),
    shipping_quote_id TEXT NOT NULL DEFAULT '',
    shipping_method_id TEXT NOT NULL DEFAULT '',
    shipping_method_name TEXT NOT NULL DEFAULT '',
    shipping_cost_cents BIGINT NOT NULL DEFAULT 0,
    destination_snapshot JSONB
);

CREATE UNIQUE INDEX orders_idempotency_key_idx ON orders (idempotency_key) WHERE idempotency_key <> '';
CREATE INDEX orders_customer_id_idx ON orders (customer_id);
CREATE INDEX orders_user_id_created_at_idx ON orders (user_id, created_at DESC, id DESC);

CREATE TABLE order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id),
    product_id TEXT NOT NULL,
    name TEXT NOT NULL,
    unit_price_cents BIGINT NOT NULL,
    quantity INT NOT NULL,
    line_total_cents BIGINT NOT NULL
);

CREATE INDEX order_items_order_id_idx ON order_items (order_id);

CREATE TABLE outbox (
    id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    message_key TEXT NOT NULL,
    body BYTEA NOT NULL,
    state INT NOT NULL,
    created_on TIMESTAMPTZ NOT NULL,
    locked_by TEXT,
    locked_on TIMESTAMPTZ,
    processed_on TIMESTAMPTZ,
    number_of_attempts INT NOT NULL DEFAULT 0,
    last_attempted_on TIMESTAMPTZ,
    error TEXT
);

CREATE INDEX outbox_pending_idx ON outbox (created_on) WHERE state = 0;
CREATE INDEX outbox_locked_idx ON outbox (locked_on) WHERE locked_by IS NOT NULL;

CREATE TABLE payments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    currency TEXT NOT NULL,
    method TEXT NOT NULL,
    status TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_ref TEXT,
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id TEXT NOT NULL DEFAULT '',
    CONSTRAINT payments_status_check CHECK (status IN ('pending', 'succeeded', 'failed', 'unknown'))
);

CREATE UNIQUE INDEX payments_idempotency_key_idx ON payments (idempotency_key);
CREATE INDEX payments_order_id_idx ON payments (order_id);

CREATE TABLE stock_reservations (
    order_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ,
    PRIMARY KEY (order_id, product_id)
);

CREATE INDEX stock_reservations_outstanding_idx ON stock_reservations (order_id) WHERE released_at IS NULL;

CREATE TABLE avatars (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL,
    bucket TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE shipping_methods (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    estimated_min_days INT NOT NULL,
    estimated_max_days INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipping_methods_days_check CHECK (estimated_min_days >= 0 AND estimated_max_days >= estimated_min_days)
);

CREATE INDEX shipping_methods_active_idx ON shipping_methods (active);

CREATE TABLE shipping_zones (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    country TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    postal_code_start TEXT NOT NULL DEFAULT '',
    postal_code_end TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX shipping_zones_country_idx ON shipping_zones (country);
CREATE INDEX shipping_zones_country_state_idx ON shipping_zones (country, state);

CREATE TABLE shipping_rates (
    id TEXT PRIMARY KEY,
    zone_id TEXT NOT NULL REFERENCES shipping_zones(id) ON DELETE CASCADE,
    method_id TEXT NOT NULL REFERENCES shipping_methods(id) ON DELETE CASCADE,
    base_cost_cents BIGINT NOT NULL,
    free_threshold_cents BIGINT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipping_rates_cost_check CHECK (base_cost_cents >= 0 AND free_threshold_cents >= 0),
    CONSTRAINT shipping_rates_unique_zone_method UNIQUE (zone_id, method_id)
);

CREATE INDEX shipping_rates_zone_idx ON shipping_rates (zone_id);
CREATE INDEX shipping_rates_method_idx ON shipping_rates (method_id);
CREATE INDEX shipping_rates_active_idx ON shipping_rates (active);

CREATE TABLE shipping_quotes (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    method_id TEXT NOT NULL REFERENCES shipping_methods(id),
    destination_snapshot JSONB NOT NULL,
    shipping_cost_cents BIGINT NOT NULL,
    estimated_min_days INT NOT NULL,
    estimated_max_days INT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    subtotal_cents BIGINT NOT NULL DEFAULT 0,
    free_threshold_cents BIGINT NOT NULL DEFAULT 0,
    idempotency_key TEXT NOT NULL DEFAULT '',
    address_id TEXT NOT NULL DEFAULT '',
    CONSTRAINT shipping_quotes_status_check CHECK (status IN ('active', 'expired', 'consumed')),
    CONSTRAINT shipping_quotes_cost_check CHECK (shipping_cost_cents >= 0),
    CONSTRAINT shipping_quotes_pricing_check CHECK (subtotal_cents >= 0 AND free_threshold_cents >= 0)
);

CREATE INDEX shipping_quotes_customer_idx ON shipping_quotes (customer_id);
CREATE INDEX shipping_quotes_status_idx ON shipping_quotes (status);
CREATE INDEX shipping_quotes_expires_at_idx ON shipping_quotes (expires_at);
CREATE INDEX shipping_quotes_customer_status_idx ON shipping_quotes (customer_id, status);
CREATE UNIQUE INDEX shipping_quotes_idempotency_key_idx ON shipping_quotes (idempotency_key) WHERE idempotency_key <> '';

CREATE TABLE shipments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    method_id TEXT NOT NULL REFERENCES shipping_methods(id),
    tracking_code TEXT NOT NULL DEFAULT '',
    carrier TEXT NOT NULL DEFAULT '',
    shipping_cost_cents BIGINT NOT NULL,
    destination_snapshot JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipments_status_check CHECK (status IN ('pending', 'processing', 'shipped', 'delivered', 'failed')),
    CONSTRAINT shipments_cost_check CHECK (shipping_cost_cents >= 0)
);

CREATE INDEX shipments_user_id_idx ON shipments (user_id);
CREATE INDEX shipments_status_idx ON shipments (status);
CREATE INDEX shipments_order_id_idx ON shipments (order_id);
CREATE INDEX shipments_user_status_idx ON shipments (user_id, status);

CREATE TABLE shipment_tracking_events (
    id TEXT PRIMARY KEY,
    shipment_id TEXT NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    description TEXT NOT NULL,
    location TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipment_tracking_events_status_check CHECK (status IN ('pending', 'processing', 'shipped', 'delivered', 'failed'))
);

CREATE INDEX shipment_tracking_events_shipment_idx ON shipment_tracking_events (shipment_id);
CREATE INDEX shipment_tracking_events_occurred_idx ON shipment_tracking_events (occurred_at);

INSERT INTO roles (id, description) VALUES
    ('customer', 'Default role for authenticated users'),
    ('support', 'Support staff with read access to all resources'),
    ('admin', 'Full administrative access')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id) VALUES
    ('catalog:read'),
    ('catalog:write'),
    ('customers:read'),
    ('customers:write'),
    ('orders:read'),
    ('orders:write'),
    ('payments:read'),
    ('payments:write'),
    ('notifications:read'),
    ('accounts:manage'),
    ('notifications:write'),
    ('accounts:write'),
    ('addresses:read'),
    ('addresses:write'),
    ('avatars:read'),
    ('avatars:write'),
    ('shipping:read'),
    ('shipping:write'),
    ('shipping:admin'),
    ('reviews:write'),
    ('cart:read'),
    ('cart:write'),
    ('wishlist:read'),
    ('wishlist:write')
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id) VALUES
    ('customer', 'catalog:read'),
    ('customer', 'orders:read'),
    ('customer', 'orders:write'),
    ('customer', 'payments:read'),
    ('customer', 'payments:write'),
    ('customer', 'notifications:read'),
    ('customer', 'customers:read'),
    ('customer', 'customers:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id) VALUES
    ('support', 'catalog:read'),
    ('support', 'orders:read'),
    ('support', 'payments:read'),
    ('support', 'notifications:read'),
    ('support', 'customers:read'),
    ('support', 'customers:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id) VALUES
    ('admin', 'catalog:read'),
    ('admin', 'catalog:write'),
    ('admin', 'customers:read'),
    ('admin', 'customers:write'),
    ('admin', 'orders:read'),
    ('admin', 'orders:write'),
    ('admin', 'payments:read'),
    ('admin', 'payments:write'),
    ('admin', 'notifications:read'),
    ('admin', 'accounts:manage')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id) VALUES
    ('customer', 'notifications:write'),
    ('support', 'notifications:write'),
    ('admin', 'notifications:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id) VALUES
    ('customer', 'accounts:write'),
    ('support', 'accounts:write'),
    ('admin', 'accounts:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id IN ('customer', 'support', 'admin')
  AND p.id IN ('addresses:read', 'addresses:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id IN ('customer', 'support', 'admin')
  AND p.id IN ('avatars:read', 'avatars:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id = 'customer' AND p.id = 'shipping:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id = 'support' AND p.id = 'shipping:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id = 'admin' AND p.id IN ('shipping:read', 'shipping:write', 'shipping:admin')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id = 'customer' AND p.id = 'reviews:write'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id = 'admin' AND p.id = 'reviews:write'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id IN ('customer', 'support', 'admin')
  AND p.id IN ('cart:read', 'cart:write', 'wishlist:read', 'wishlist:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO shipping_methods (id, name, description, active, estimated_min_days, estimated_max_days) VALUES
    ('sm_std', 'Standard Delivery', 'Standard shipping 3-5 business days', TRUE, 3, 5),
    ('sm_exp', 'Express Delivery', 'Express shipping 1-2 business days', TRUE, 1, 2)
ON CONFLICT (id) DO NOTHING;

INSERT INTO shipping_zones (id, name, country, state, city, postal_code_start, postal_code_end) VALUES
    ('sz_br', 'Brazil', 'BR', '', '', '', '')
ON CONFLICT (id) DO NOTHING;

INSERT INTO shipping_rates (id, zone_id, method_id, base_cost_cents, free_threshold_cents, active) VALUES
    ('sr_std_br', 'sz_br', 'sm_std', 599, 5000, TRUE),
    ('sr_exp_br', 'sz_br', 'sm_exp', 1299, 10000, TRUE)
ON CONFLICT (id) DO NOTHING;

-- migrate:down

DROP TABLE IF EXISTS shipment_tracking_events;
DROP TABLE IF EXISTS shipments;
DROP TABLE IF EXISTS shipping_quotes;
DROP TABLE IF EXISTS shipping_rates;
DROP TABLE IF EXISTS shipping_zones;
DROP TABLE IF EXISTS shipping_methods;
DROP TABLE IF EXISTS avatars;
DROP TABLE IF EXISTS stock_reservations;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS product_reviews;
DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS role_permission;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
