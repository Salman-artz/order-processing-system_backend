-- ==============================================================
-- migrations: 000002_warehouseflow_payment_lifecycle.up.sql
-- Service  : payment-service
-- Database : payment_db
-- ==============================================================

ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS provider VARCHAR(32) NOT NULL DEFAULT 'sandbox',
    ADD COLUMN IF NOT EXISTS transaction_id VARCHAR(128) NULL,
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(128) NULL,
    ADD COLUMN IF NOT EXISTS signature VARCHAR(255) NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_idempotency_key 
    ON payments(idempotency_key) WHERE idempotency_key IS NOT NULL;

-- Processed events table for idempotent payment event consumers
CREATE TABLE IF NOT EXISTS processed_events (
    event_id      VARCHAR(128) PRIMARY KEY,
    event_type    VARCHAR(64)  NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_type ON processed_events(event_type);
