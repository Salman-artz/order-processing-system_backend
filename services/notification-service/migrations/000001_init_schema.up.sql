-- ==============================================================
-- migrations: 000001_init_schema.up.sql
-- Service  : notification-service
-- Database : notification_db
-- ==============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Table: notifications_log ──────────────────────────────────
CREATE TABLE IF NOT EXISTS notifications_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID        NOT NULL,
    type        VARCHAR(64) NOT NULL,
                -- e.g. ORDER_CREATED, STOCK_RESERVED, PAYMENT_SUCCESS,
                --      PAYMENT_FAILED, ORDER_COMPLETED, ORDER_CANCELLED
    channel     VARCHAR(32) NOT NULL DEFAULT 'email_mock',
                -- email_mock | websocket
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Indexes ───────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_notifications_log_order_id ON notifications_log(order_id);
CREATE INDEX IF NOT EXISTS idx_notifications_log_type     ON notifications_log(type);
CREATE INDEX IF NOT EXISTS idx_notifications_log_sent_at  ON notifications_log(sent_at DESC);

-- ── Auto-update updated_at trigger ───────────────────────────
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_notifications_log_updated_at
    BEFORE UPDATE ON notifications_log
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
