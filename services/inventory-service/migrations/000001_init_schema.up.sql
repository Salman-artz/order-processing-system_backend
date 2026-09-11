-- ==============================================================
-- migrations: 000001_init_schema.up.sql
-- Service  : inventory-service
-- Database : inventory_db
-- ==============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Table: products ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS products (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(255)   NOT NULL,
    stock_available  INT            NOT NULL DEFAULT 0,
    stock_reserved   INT            NOT NULL DEFAULT 0,
    price            DECIMAL(15, 2) NOT NULL CHECK (price >= 0),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),

    -- Stok tidak boleh negatif
    CONSTRAINT products_stock_available_non_negative CHECK (stock_available >= 0),
    CONSTRAINT products_stock_reserved_non_negative  CHECK (stock_reserved >= 0)
);

-- ── Table: stock_reservations ─────────────────────────────────
CREATE TABLE IF NOT EXISTS stock_reservations (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID        NOT NULL,
    product_id  UUID        NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity    INT         NOT NULL CHECK (quantity > 0),
    status      VARCHAR(16) NOT NULL DEFAULT 'RESERVED',
                -- RESERVED | RELEASED | CONFIRMED
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT stock_reservations_status_check CHECK (
        status IN ('RESERVED', 'RELEASED', 'CONFIRMED')
    )
);

-- ── Indexes ───────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_products_name            ON products(name);

CREATE INDEX IF NOT EXISTS idx_stock_reservations_order_id   ON stock_reservations(order_id);
CREATE INDEX IF NOT EXISTS idx_stock_reservations_product_id ON stock_reservations(product_id);
CREATE INDEX IF NOT EXISTS idx_stock_reservations_status     ON stock_reservations(status);

-- ── Auto-update updated_at trigger ───────────────────────────
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_stock_reservations_updated_at
    BEFORE UPDATE ON stock_reservations
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
