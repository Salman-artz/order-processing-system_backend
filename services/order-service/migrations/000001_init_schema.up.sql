-- ==============================================================
-- migrations: 000001_init_schema.up.sql
-- Service  : order-service
-- Database : order_db
-- ==============================================================

-- Enable pgcrypto untuk gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Table: orders ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS orders (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id     UUID        NOT NULL,
    status          VARCHAR(32) NOT NULL DEFAULT 'PENDING',
                    -- PENDING | AWAITING_PAYMENT | COMPLETED | CANCELLED
    total_amount    DECIMAL(15, 2) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT orders_status_check CHECK (
        status IN ('PENDING', 'AWAITING_PAYMENT', 'COMPLETED', 'CANCELLED')
    )
);

-- ── Table: order_items ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS order_items (
    id          UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID           NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id  UUID           NOT NULL,
    quantity    INT            NOT NULL CHECK (quantity > 0),
    price       DECIMAL(15, 2) NOT NULL CHECK (price >= 0),
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT now()
);

-- ── Table: saga_logs ──────────────────────────────────────────
-- Audit trail & debugging untuk setiap event dalam saga flow
CREATE TABLE IF NOT EXISTS saga_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID        NOT NULL,   -- soft ref, tidak FK agar log tetap ada walau order terhapus
    event_type  VARCHAR(64) NOT NULL,   -- e.g. order.created, stock.reserved, payment.success
    payload     JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
    -- tidak ada updated_at: saga_logs bersifat append-only / immutable
);

-- ── Indexes ───────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_status      ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at  ON orders(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);

CREATE INDEX IF NOT EXISTS idx_saga_logs_order_id   ON saga_logs(order_id);
CREATE INDEX IF NOT EXISTS idx_saga_logs_event_type ON saga_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_saga_logs_created_at ON saga_logs(created_at DESC);

-- ── Auto-update updated_at trigger ───────────────────────────
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_order_items_updated_at
    BEFORE UPDATE ON order_items
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
