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
CREATE TABLE IF NOT EXISTS drivers (
	id SERIAL PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

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

-- ALTER .. IF NOT EXISTS instead of baking this into CREATE TABLE above, so it
-- applies cleanly to databases that already had the trips table before this column existed.
ALTER TABLE trips ADD COLUMN IF NOT EXISTS next_rest_suggestion_min DOUBLE PRECISION;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS rest_stop_lat DOUBLE PRECISION;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS rest_stop_lon DOUBLE PRECISION;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS rest_stop_name TEXT;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS rest_stop_amenity TEXT;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS explanation TEXT;
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS driver_id INTEGER REFERENCES drivers(id);
ALTER TABLE trips ADD COLUMN IF NOT EXISTS driver_id INTEGER REFERENCES drivers(id);

CREATE TABLE IF NOT EXISTS driver_preferences (
	driver_id INTEGER PRIMARY KEY REFERENCES drivers(id),
	fuel_priority SMALLINT NOT NULL DEFAULT 3 CHECK (fuel_priority BETWEEN 1 AND 5),
	cargo_priority SMALLINT NOT NULL DEFAULT 3 CHECK (cargo_priority BETWEEN 1 AND 5),
	highway_priority SMALLINT NOT NULL DEFAULT 3 CHECK (highway_priority BETWEEN 1 AND 5),
	time_priority SMALLINT NOT NULL DEFAULT 3 CHECK (time_priority BETWEEN 1 AND 5),
	preferred_fuel_brand TEXT
);

CREATE TABLE IF NOT EXISTS driver_favorite_stops (
	id SERIAL PRIMARY KEY,
	driver_id INTEGER NOT NULL REFERENCES drivers(id),
	lat DOUBLE PRECISION NOT NULL,
	lon DOUBLE PRECISION NOT NULL,
	name TEXT,
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
