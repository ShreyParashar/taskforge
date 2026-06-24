package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"taskforge/internal/config"
)

// Store encapsulates the database connection pool and provides access
// to all database operations for TaskForge.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore initializes a new PostgreSQL connection pool and returns a Store.
func NewStore(ctx context.Context, cfg config.DatabaseConfig) (*Store, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// Set sensible connection pool defaults for a high-concurrency worker engine
	poolConfig.MaxConns = 50
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &Store{
		pool: pool,
	}, nil
}

// Close closes the database connection pool.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// Ping checks the database connection.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
