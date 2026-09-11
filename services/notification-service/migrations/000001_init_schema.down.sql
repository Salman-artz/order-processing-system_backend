-- ==============================================================
-- migrations: 000001_init_schema.down.sql
-- Service  : notification-service
-- Database : notification_db
-- ==============================================================

DROP TRIGGER IF EXISTS set_notifications_log_updated_at ON notifications_log;
DROP FUNCTION IF EXISTS trigger_set_updated_at();

DROP INDEX IF EXISTS idx_notifications_log_sent_at;
DROP INDEX IF EXISTS idx_notifications_log_type;
DROP INDEX IF EXISTS idx_notifications_log_order_id;

DROP TABLE IF EXISTS notifications_log;

DROP EXTENSION IF EXISTS "pgcrypto";
