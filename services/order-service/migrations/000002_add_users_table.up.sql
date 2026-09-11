-- ==============================================================
-- migrations: 000002_add_users_table.up.sql
-- Service  : order-service
-- Database : order_db
-- ==============================================================

CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(32)  NOT NULL DEFAULT 'customer',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT users_role_check CHECK (role IN ('customer', 'admin'))
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Add foreign key from orders to users for data integrity
ALTER TABLE orders 
    ADD CONSTRAINT fk_orders_customer_id 
    FOREIGN KEY (customer_id) REFERENCES users(id) ON DELETE RESTRICT;
