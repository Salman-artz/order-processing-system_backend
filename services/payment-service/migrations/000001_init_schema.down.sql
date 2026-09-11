-- ==============================================================
-- migrations: 000001_init_schema.down.sql
-- Service  : payment-service
-- Database : payment_db
-- ==============================================================

DROP TRIGGER IF EXISTS set_payments_updated_at ON payments;
DROP FUNCTION IF EXISTS trigger_set_updated_at();

DROP INDEX IF EXISTS idx_payments_created_at;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_order_id;

DROP TABLE IF EXISTS payments;

DROP EXTENSION IF EXISTS "pgcrypto";
