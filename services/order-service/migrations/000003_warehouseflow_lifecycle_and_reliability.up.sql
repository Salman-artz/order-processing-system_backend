-- ==============================================================
-- migrations: 000003_warehouseflow_lifecycle_and_reliability.up.sql
-- Service  : order-service
-- Database : order_db
-- ==============================================================

-- 1. Upgrade orders status constraint to support full WarehouseFlow fulfillment lifecycle
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;

ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (
    status IN (
        'PENDING',
        'INVENTORY_RESERVED',
        'PAYMENT_PENDING',
        'CONFIRMED',
        'PICKING',
        'PACKED',
        'DISPATCHED',
        'COMPLETED',
        'PAYMENT_FAILED',
        'CANCELLED',
        'COMPENSATION_PENDING',
        'COMPENSATION_FAILED'
    )
);

-- 2. Transactional Outbox Table for reliable event publishing
CREATE TABLE IF NOT EXISTS outbox (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type    VARCHAR(64)  NOT NULL,
    payload       JSONB        NOT NULL,
    status        VARCHAR(16)  NOT NULL DEFAULT 'PENDING', -- PENDING | PUBLISHED | FAILED
    retry_count   INT          NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    published_at  TIMESTAMPTZ  NULL,

    CONSTRAINT outbox_status_check CHECK (status IN ('PENDING', 'PUBLISHED', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_outbox_status_created ON outbox(status, created_at ASC);

-- 3. Idempotency Inbox / Processed Events Table to prevent duplicate processing
CREATE TABLE IF NOT EXISTS processed_events (
    event_id      VARCHAR(128) PRIMARY KEY,
    event_type    VARCHAR(64)  NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_type ON processed_events(event_type);
