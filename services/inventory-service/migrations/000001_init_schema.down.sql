-- ==============================================================
-- migrations: 000001_init_schema.down.sql
-- Service  : inventory-service
-- Database : inventory_db
-- ==============================================================

DROP TRIGGER IF EXISTS set_stock_reservations_updated_at ON stock_reservations;
DROP TRIGGER IF EXISTS set_products_updated_at ON products;

DROP FUNCTION IF EXISTS trigger_set_updated_at();

DROP INDEX IF EXISTS idx_stock_reservations_status;
DROP INDEX IF EXISTS idx_stock_reservations_product_id;
DROP INDEX IF EXISTS idx_stock_reservations_order_id;
DROP INDEX IF EXISTS idx_products_name;

DROP TABLE IF EXISTS stock_reservations;
DROP TABLE IF EXISTS products;

DROP EXTENSION IF EXISTS "pgcrypto";
