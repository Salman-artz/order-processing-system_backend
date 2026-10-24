DROP TABLE IF EXISTS processed_events;

ALTER TABLE payments
    DROP COLUMN IF EXISTS signature,
    DROP COLUMN IF EXISTS idempotency_key,
    DROP COLUMN IF EXISTS transaction_id,
    DROP COLUMN IF EXISTS provider;
