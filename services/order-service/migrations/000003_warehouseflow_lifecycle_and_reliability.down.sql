DROP TABLE IF EXISTS processed_events;
DROP TABLE IF EXISTS outbox;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (
    status IN ('PENDING', 'AWAITING_PAYMENT', 'COMPLETED', 'CANCELLED')
);
