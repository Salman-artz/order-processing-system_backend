-- ==============================================================
-- migrations: 000001_init_schema.up.sql
-- Service  : payment-service
-- Database : payment_db
-- ==============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Table: payments ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS payments (
    id                        UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id                  UUID           NOT NULL,
    amount                    DECIMAL(15, 2) NOT NULL CHECK (amount > 0),
    status                    VARCHAR(16)    NOT NULL DEFAULT 'PENDING',
                              -- PENDING | SUCCESS | FAILED
    simulated_failure_reason  VARCHAR(255)   NULL,  -- diisi jika status = FAILED
    created_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT payments_status_check CHECK (
        status IN ('PENDING', 'SUCCESS', 'FAILED')
    )
);

-- ── Indexes ───────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_payments_order_id   ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_status     ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);

-- ── Auto-update updated_at trigger ───────────────────────────
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
