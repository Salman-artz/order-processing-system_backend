-- ==============================================================
-- scripts/create-databases.sql
-- ==============================================================
-- CARA MENJALANKAN:
--   Opsi A (psql CLI):
--     psql -U <user> -h localhost -f scripts/create-databases.sql
--
--   Opsi B (pgAdmin / TablePlus):
--     Buka Query Editor, paste isi file ini, lalu Execute.
--
-- CATATAN: Script ini harus dijalankan sebagai superuser PostgreSQL
-- (biasanya user "postgres"). Jika koneksi kamu sudah sebagai
-- superuser, langsung jalankan saja.
-- ==============================================================

-- Buat 4 database terpisah untuk setiap microservice
-- (Abaikan error "already exists" jika database sudah ada)

SELECT 'Creating order_db...' AS status;
CREATE DATABASE order_db
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8'
    TEMPLATE = template0;

COMMENT ON DATABASE order_db IS 'Order Processing System — Order Service database';

SELECT 'Creating inventory_db...' AS status;
CREATE DATABASE inventory_db
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8'
    TEMPLATE = template0;

COMMENT ON DATABASE inventory_db IS 'Order Processing System — Inventory Service database';

SELECT 'Creating payment_db...' AS status;
CREATE DATABASE payment_db
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8'
    TEMPLATE = template0;

COMMENT ON DATABASE payment_db IS 'Order Processing System — Payment Service database';

SELECT 'Creating notification_db...' AS status;
CREATE DATABASE notification_db
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8'
    TEMPLATE = template0;

COMMENT ON DATABASE notification_db IS 'Order Processing System — Notification Service database';

SELECT '✅ Semua database berhasil dibuat!' AS status;

-- Verifikasi: tampilkan semua database yang baru dibuat
SELECT datname AS "Database Name", pg_encoding_to_char(encoding) AS "Encoding"
FROM pg_database
WHERE datname IN ('order_db', 'inventory_db', 'payment_db', 'notification_db')
ORDER BY datname;
