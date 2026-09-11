-- ==============================================================
-- migrations/seed.sql
-- Service  : inventory-service
-- Database : inventory_db
--
-- CARA MENJALANKAN (setelah migrate):
--   psql -U <user> -h localhost -d inventory_db -f services/inventory-service/migrations/seed.sql
--
-- Atau lewat pgAdmin/TablePlus: pilih database inventory_db,
-- paste isi file ini di query editor, lalu execute.
-- ==============================================================

-- Bersihkan data lama jika ada (opsional, hanya untuk re-seed)
-- TRUNCATE TABLE stock_reservations, products RESTART IDENTITY CASCADE;

INSERT INTO products (id, name, stock_available, stock_reserved, price)
VALUES
    -- Electronics
    (gen_random_uuid(), 'Laptop ASUS VivoBook 15 Intel Core i5',       25,  0, 8750000.00),
    (gen_random_uuid(), 'Wireless Mouse Logitech MX Master 3',         80,  0,  950000.00),
    (gen_random_uuid(), 'Mechanical Keyboard Keychron K2 RGB',         40,  0, 1350000.00),
    (gen_random_uuid(), 'Monitor LG 24 inch IPS Full HD',              15,  0, 2850000.00),

    -- Accessories & Gadgets
    (gen_random_uuid(), 'USB-C Hub 7-in-1 Anker Premium',              60,  0,  425000.00),
    (gen_random_uuid(), 'Webcam Logitech C920 HD Pro',                 30,  0, 1200000.00),
    (gen_random_uuid(), 'Headphone Sony WH-1000XM5 Noise Cancelling',  20,  0, 4500000.00),
    (gen_random_uuid(), 'SSD External Samsung T7 1TB',                 35,  0, 1750000.00),

    -- Low stock items (untuk testing saga stock.rejected scenario)
    (gen_random_uuid(), 'GPU NVIDIA RTX 4090 24GB',                     2,  0, 28500000.00),
    (gen_random_uuid(), 'Standing Desk Ergonomic Motorized 140cm',      5,  0, 5200000.00)

ON CONFLICT (id) DO NOTHING;

-- Verifikasi seed
SELECT
    name,
    stock_available,
    stock_reserved,
    TO_CHAR(price, 'FM999,999,999,999') AS price_idr
FROM products
ORDER BY price DESC;
