// Package db contains connection helpers and (in later phases) the generated
// sqlc data-access code for PostgreSQL.
//
// Phase 1 only wires up the connection pool. Queries and repositories live in
// the feature packages (internal/users, internal/events, internal/bookings).
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgres creates a pgx connection pool for the given database URL and
// validates connectivity with a ping. The returned pool must be closed by the
// caller (typically via defer) on shutdown.
func NewPostgres(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	// Sensible defaults for a modular monolith. These can be tuned via the
	// DATABASE_URL once we measure under load (see k6 plan in README).
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
