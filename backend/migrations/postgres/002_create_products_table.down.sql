-- Migration: 002_create_products_table
-- Direction: DOWN

BEGIN;
DROP TRIGGER IF EXISTS orders_updated_at  ON orders;
DROP TRIGGER IF EXISTS products_updated_at ON products;
DROP TABLE   IF EXISTS order_items;
DROP TABLE   IF EXISTS orders;
DROP TABLE   IF EXISTS products;
COMMIT;
