DROP TABLE IF EXISTS processed_events;

ALTER TABLE notifications_log
    DROP COLUMN IF EXISTS retry_count,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS subject,
    DROP COLUMN IF EXISTS recipient;
