-- ==============================================================
-- migrations: 000002_warehouseflow_delivery_tracking.up.sql
-- Service  : notification-service
-- Database : notification_db
-- ==============================================================

ALTER TABLE notifications_log
    ADD COLUMN IF NOT EXISTS recipient VARCHAR(255) DEFAULT '',
    ADD COLUMN IF NOT EXISTS subject VARCHAR(255) DEFAULT '',
    ADD COLUMN IF NOT EXISTS status VARCHAR(32) DEFAULT 'DELIVERED',
    ADD COLUMN IF NOT EXISTS retry_count INT DEFAULT 0;

CREATE TABLE IF NOT EXISTS processed_events (
    event_id      VARCHAR(128) PRIMARY KEY,
    event_type    VARCHAR(64)  NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_type ON processed_events(event_type);
