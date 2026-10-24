// Package outbox provides reliable transactional outbox and idempotency
// primitives for PostgreSQL and NATS JetStream within WarehouseFlow.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Record holds an outbox record awaiting publication.
type Record struct {
	ID         uuid.UUID       `json:"id"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload"`
	Status     string          `json:"status"` // PENDING | PUBLISHED | FAILED
	RetryCount int             `json:"retry_count"`
	CreatedAt  time.Time       `json:"created_at"`
}

// DBExecutor represents an interface satisfied by both *pgxpool.Pool and pgx.Tx.
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
}

// Save records an outbox message inside an ongoing database transaction.
func Save(ctx context.Context, exec DBExecutor, eventType string, payload any) (uuid.UUID, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("outbox: marshal payload: %w", err)
	}

	id := uuid.New()
	query := `
		INSERT INTO outbox (id, event_type, payload, status, retry_count, created_at)
		VALUES ($1, $2, $3, 'PENDING', 0, now())
		RETURNING id
	`
	var outID uuid.UUID
	err = exec.QueryRow(ctx, query, id, eventType, bytes).Scan(&outID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("outbox: insert record: %w", err)
	}
	return outID, nil
}

// MarkPublished marks an outbox record as PUBLISHED.
func MarkPublished(ctx context.Context, exec DBExecutor, id uuid.UUID) error {
	query := `
		UPDATE outbox
		SET status = 'PUBLISHED', published_at = now()
		WHERE id = $1
	`
	_, err := exec.Exec(ctx, query, id)
	return err
}

// MarkFailed increments retry_count and marks status as FAILED if maxRetries exceeded.
func MarkFailed(ctx context.Context, exec DBExecutor, id uuid.UUID, maxRetries int) error {
	query := `
		UPDATE outbox
		SET retry_count = retry_count + 1,
		    status = CASE WHEN retry_count + 1 >= $2 THEN 'FAILED' ELSE 'PENDING' END
		WHERE id = $1
	`
	_, err := exec.Exec(ctx, query, id, maxRetries)
	return err
}

// GetPending fetches batch of unpublished records.
func GetPending(ctx context.Context, pool *pgxpool.Pool, limit int) ([]Record, error) {
	query := `
		SELECT id, event_type, payload, status, retry_count, created_at
		FROM outbox
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
	`
	rows, err := pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.ID, &r.EventType, &r.Payload, &r.Status, &r.RetryCount, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ─── Idempotent Consumer Helpers ──────────────────────────────────────────────

// CheckAndRecordProcessed checks if an event has already been processed.
// If it has NOT been processed, it registers the event in processed_events and returns (false, nil).
// If it HAS already been processed (duplicate delivery), it returns (true, nil).
func CheckAndRecordProcessed(ctx context.Context, exec DBExecutor, eventID string, eventType string) (bool, error) {
	if eventID == "" {
		return false, nil // un-identified event; skip check
	}

	query := `
		INSERT INTO processed_events (event_id, event_type, processed_at)
		VALUES ($1, $2, now())
		ON CONFLICT (event_id) DO NOTHING
	`
	tag, err := exec.Exec(ctx, query, eventID, eventType)
	if err != nil {
		return false, fmt.Errorf("outbox: record processed event: %w", err)
	}

	// If 0 rows affected, it already existed (duplicate!)
	if tag.RowsAffected() == 0 {
		return true, nil
	}
	return false, nil
}

// IsProcessed checks if event was already marked as processed.
func IsProcessed(ctx context.Context, exec DBExecutor, eventID string) (bool, error) {
	if eventID == "" {
		return false, nil
	}

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)`
	err := exec.QueryRow(ctx, query, eventID).Scan(&exists)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return exists, nil
}
