package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for database/sql
	"github.com/pressly/goose/v3"

	"taskforge/internal/config"
)

// MigrateUp applies all pending database migrations programmatically.
// It uses database/sql with pgx stdlib because goose requires a *sql.DB.
func MigrateUp(cfg config.DatabaseConfig, dir string) error {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database for migrations: %w", err)
	}

	// Disable goose output for cleaner startup logs
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}