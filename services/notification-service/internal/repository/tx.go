package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository extends the generated Querier interface with transaction support
type Repository interface {
	Querier
	WithTx(ctx context.Context, fn func(q Querier) error) error
}

type repo struct {
	*Queries
	db *pgxpool.Pool
}

// NewRepository returns a new Repository instance
func NewRepository(db *pgxpool.Pool) Repository {
	return &repo{
		Queries: New(db),
		db:      db,
	}
}

// WithTx executes the given function within a database transaction.
func (r *repo) WithTx(ctx context.Context, fn func(q Querier) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.Queries.WithTx(tx)
	if err := fn(qtx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
			return fmt.Errorf("repository: tx rollback err: %v, orig err: %w", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: commit tx: %w", err)
	}
	return nil
}
