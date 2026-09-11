-- ==============================================================
-- migrations: 000001_init_schema.down.sql
-- Service  : order-service
-- Database : order_db
-- ==============================================================

-- Drop triggers first
DROP TRIGGER IF EXISTS set_order_items_updated_at ON order_items;
DROP TRIGGER IF EXISTS set_orders_updated_at ON orders;

-- Drop function
DROP FUNCTION IF EXISTS trigger_set_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_saga_logs_created_at;
DROP INDEX IF EXISTS idx_saga_logs_event_type;
DROP INDEX IF EXISTS idx_saga_logs_order_id;
DROP INDEX IF EXISTS idx_order_items_order_id;
DROP INDEX IF EXISTS idx_orders_created_at;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_customer_id;

-- Drop tables (order matters due to FK)
DROP TABLE IF EXISTS saga_logs;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;

-- Drop extension
DROP EXTENSION IF EXISTS "pgcrypto";
