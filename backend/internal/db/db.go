// Package db owns the Postgres connection and schema migrations (via goose -
// see internal/db/migrations). Each schema change is its own numbered .sql
// file rather than one growing string, so `git log` on a single migration
// file is a clean history of exactly what changed and when.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Connect(ctx context.Context, databaseURL string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return conn, nil
}

// Migrate applies any not-yet-applied migrations. It's safe to call on every
// startup - goose tracks applied versions in its own goose_db_version table.
func Migrate(ctx context.Context, conn *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(conn, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
