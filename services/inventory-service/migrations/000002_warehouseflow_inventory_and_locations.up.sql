-- ==============================================================
-- migrations: 000002_warehouseflow_inventory_and_locations.up.sql
-- Service  : inventory-service
-- Database : inventory_db
-- ==============================================================

-- 1. Extend products table with WarehouseFlow metadata
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS sku VARCHAR(64),
    ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '',
    ADD COLUMN IF NOT EXISTS category VARCHAR(64) DEFAULT 'General Hardware',
    ADD COLUMN IF NOT EXISTS unit_of_measure VARCHAR(32) DEFAULT 'unit',
    ADD COLUMN IF NOT EXISTS cost_price DECIMAL(15, 2) DEFAULT 0 CHECK (cost_price >= 0),
    ADD COLUMN IF NOT EXISTS min_stock_threshold INT DEFAULT 5 CHECK (min_stock_threshold >= 0),
    ADD COLUMN IF NOT EXISTS status VARCHAR(16) DEFAULT 'active';

-- Populate SKUs for existing products if NULL
UPDATE products 
SET sku = 'SKU-' || UPPER(SUBSTRING(REPLACE(name, ' ', '') FROM 1 FOR 4)) || '-' || SUBSTRING(id::text FROM 1 FOR 4)
WHERE sku IS NULL;

-- Now enforce NOT NULL and UNIQUE on sku
ALTER TABLE products ALTER COLUMN sku SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_products_sku ON products(sku);

-- 2. Warehouse Locations Table (Aisle/Zone, Rack, Shelf, Bin)
CREATE TABLE IF NOT EXISTS warehouse_locations (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_name  VARCHAR(64)  NOT NULL DEFAULT 'Main Fulfillment Hub',
    zone            VARCHAR(16)  NOT NULL,  -- e.g. ZONE-A (Electronics), ZONE-B (Accessories)
    rack            VARCHAR(16)  NOT NULL,  -- e.g. RACK-01
    shelf           VARCHAR(16)  NOT NULL,  -- e.g. SHELF-03
    bin             VARCHAR(16)  NOT NULL,  -- e.g. BIN-02
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_warehouse_location UNIQUE (warehouse_name, zone, rack, shelf, bin)
);

CREATE INDEX IF NOT EXISTS idx_warehouse_locations_zone ON warehouse_locations(zone);

-- 3. Stock Movements Table (Auditable ledger: Receiving, Issuing, Adjustment, Reservation, Release, Transfer)
CREATE TABLE IF NOT EXISTS stock_movements (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id          UUID         NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    location_id         UUID         NULL REFERENCES warehouse_locations(id) ON DELETE SET NULL,
    movement_type       VARCHAR(32)  NOT NULL, 
                        -- INBOUND_RECEIVING | OUTBOUND_DISPATCH | MANUAL_ADJUSTMENT | SAGA_RESERVE | SAGA_RELEASE | SAGA_CONFIRM | TRANSFER
    quantity            INT          NOT NULL,
    reference_id        VARCHAR(128) NULL,     -- Order ID, Batch ID, PO number, or Adjustment Reason
    previous_available  INT          NOT NULL,
    new_available       INT          NOT NULL,
    reason              TEXT         NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT stock_movements_type_check CHECK (
        movement_type IN (
            'INBOUND_RECEIVING',
            'OUTBOUND_DISPATCH',
            'MANUAL_ADJUSTMENT',
            'SAGA_RESERVE',
            'SAGA_RELEASE',
            'SAGA_CONFIRM',
            'TRANSFER'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_product_id ON stock_movements(product_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_created_at ON stock_movements(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_stock_movements_type       ON stock_movements(movement_type);

-- 4. Transactional Outbox for Inventory Service
CREATE TABLE IF NOT EXISTS outbox (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type    VARCHAR(64)  NOT NULL,
    payload       JSONB        NOT NULL,
    status        VARCHAR(16)  NOT NULL DEFAULT 'PENDING',
    retry_count   INT          NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    published_at  TIMESTAMPTZ  NULL,

    CONSTRAINT outbox_status_check CHECK (status IN ('PENDING', 'PUBLISHED', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_outbox_status_created ON outbox(status, created_at ASC);

-- 5. Processed Events table for Idempotent Consumer
CREATE TABLE IF NOT EXISTS processed_events (
    event_id      VARCHAR(128) PRIMARY KEY,
    event_type    VARCHAR(64)  NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_type ON processed_events(event_type);

-- Seed default warehouse locations
INSERT INTO warehouse_locations (warehouse_name, zone, rack, shelf, bin)
VALUES
    ('Main Fulfillment Hub', 'ZONE-A', 'R01', 'S01', 'B01'),
    ('Main Fulfillment Hub', 'ZONE-A', 'R01', 'S01', 'B02'),
    ('Main Fulfillment Hub', 'ZONE-A', 'R01', 'S02', 'B01'),
    ('Main Fulfillment Hub', 'ZONE-B', 'R02', 'S01', 'B01'),
    ('Main Fulfillment Hub', 'ZONE-B', 'R02', 'S02', 'B01'),
    ('Main Fulfillment Hub', 'ZONE-C', 'R03', 'S01', 'B01')
ON CONFLICT (warehouse_name, zone, rack, shelf, bin) DO NOTHING;
