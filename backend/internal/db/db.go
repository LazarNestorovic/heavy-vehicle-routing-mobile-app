// Package db owns the Postgres connection and the (deliberately minimal, no
// external migration tool) schema for this MVP.
package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const schema = `
CREATE TABLE IF NOT EXISTS vehicles (
	id SERIAL PRIMARY KEY,
	height_m DOUBLE PRECISION NOT NULL,
	width_m DOUBLE PRECISION NOT NULL,
	length_m DOUBLE PRECISION NOT NULL,
	weight_kg DOUBLE PRECISION NOT NULL,
	axle_load_kg DOUBLE PRECISION NOT NULL,
	hazmat BOOLEAN NOT NULL DEFAULT false,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	CHECK (axle_load_kg <= weight_kg)
);

CREATE TABLE IF NOT EXISTS trips (
	id SERIAL PRIMARY KEY,
	vehicle_id INTEGER NOT NULL REFERENCES vehicles(id),
	origin_lat DOUBLE PRECISION NOT NULL,
	origin_lon DOUBLE PRECISION NOT NULL,
	destination_lat DOUBLE PRECISION NOT NULL,
	destination_lon DOUBLE PRECISION NOT NULL,
	distance_km DOUBLE PRECISION NOT NULL,
	duration_min DOUBLE PRECISION NOT NULL,
	risk_score DOUBLE PRECISION NOT NULL,
	shape TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'created',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

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

// Migrate applies the schema. It's idempotent (CREATE TABLE IF NOT EXISTS),
// so it's safe to call on every startup instead of running a separate migration tool.
func Migrate(ctx context.Context, conn *sql.DB) error {
	if _, err := conn.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
