-- ==============================================================
-- migrations: 000002_add_users_table.down.sql
-- Service  : order-service
-- Database : order_db
-- ==============================================================

ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_orders_customer_id;

DROP TRIGGER IF EXISTS set_users_updated_at ON users;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
