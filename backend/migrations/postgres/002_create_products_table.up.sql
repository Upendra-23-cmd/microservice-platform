-- Migration: 002_create_products_table
-- Direction: UP
-- Description: Creates the products table. Rich/flexible metadata is stored in MongoDB.

BEGIN;

-- ============================================================
-- PRODUCTS TABLE (Relational core — price, stock, category)
-- ============================================================
CREATE TABLE IF NOT EXISTS products (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255)  NOT NULL,
    description TEXT,
    price       NUMERIC(12,2) NOT NULL CHECK (price > 0),
    stock       INTEGER       NOT NULL DEFAULT 0 CHECK (stock >= 0),
    category    VARCHAR(100)  NOT NULL DEFAULT 'uncategorized',
    is_active   BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS products_category_idx   ON products (category)  WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS products_price_idx      ON products (price)     WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS products_is_active_idx  ON products (is_active) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS products_created_at_idx ON products (created_at);

-- Full-text search on name + description
CREATE INDEX IF NOT EXISTS products_search_idx
    ON products USING GIN (
        to_tsvector('english', name || ' ' || COALESCE(description, ''))
    )
    WHERE deleted_at IS NULL;

-- Stock range index (useful for "in stock" filters)
CREATE INDEX IF NOT EXISTS products_stock_idx ON products (stock) WHERE deleted_at IS NULL;

-- Auto-update trigger (reuse function from users migration)
CREATE TRIGGER products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- ORDERS TABLE (core relational reference)
-- ============================================================
CREATE TABLE IF NOT EXISTS orders (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status      VARCHAR(30) NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending','confirmed','processing','shipped','delivered','cancelled','refunded')),
    total       NUMERIC(12,2) NOT NULL CHECK (total >= 0),
    currency    VARCHAR(3)  NOT NULL DEFAULT 'USD',
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS orders_user_id_idx   ON orders (user_id)   WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS orders_status_idx    ON orders (status)    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS orders_created_at_idx ON orders (created_at);

CREATE TRIGGER orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- ORDER_ITEMS TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS order_items (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id  UUID          NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity    INTEGER       NOT NULL CHECK (quantity > 0),
    unit_price  NUMERIC(12,2) NOT NULL CHECK (unit_price > 0),
    subtotal    NUMERIC(12,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS order_items_order_id_idx   ON order_items (order_id);
CREATE INDEX IF NOT EXISTS order_items_product_id_idx ON order_items (product_id);

COMMIT;
