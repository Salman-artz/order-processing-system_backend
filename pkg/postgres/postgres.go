// Package postgres provides a shared PostgreSQL connection helper using pgx/v5.
// Each service calls postgres.Connect() in main() to get a *pgxpool.Pool.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database connection configuration.
type Config struct {
	// DSN is the full PostgreSQL connection string.
	// Example: postgres://user:pass@localhost:5432/order_db?sslmode=disable
	DSN string

	// Pool tuning (sensible defaults if zero)
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

func (c *Config) applyDefaults() {
	if c.MaxConns == 0 {
		c.MaxConns = 25
	}
	if c.MinConns == 0 {
		c.MinConns = 5
	}
	if c.MaxConnLifetime == 0 {
		c.MaxConnLifetime = 5 * time.Minute
	}
	if c.MaxConnIdleTime == 0 {
		c.MaxConnIdleTime = 1 * time.Minute
	}
}

// Connect establishes a connection pool to PostgreSQL and pings it.
// Returns an error if the database is unreachable.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	cfg.applyDefaults()

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping failed: %w", err)
	}

	return pool, nil
}

// MustConnect is like Connect but panics on error.
// Suitable for use in main() where a DB connection is mandatory.
func MustConnect(ctx context.Context, cfg Config) *pgxpool.Pool {
	pool, err := Connect(ctx, cfg)
	if err != nil {
		fmt.Printf("FATAL ERROR: %v\n", err); panic(fmt.Sprintf("postgres: fatal connection error: %v", err))
	}
	return pool
}
