-- +goose Up
-- Bootstrap migration: this is the full schema as it existed before the
-- goose migration system was introduced (see documentations/features/ entry
-- for the dispatcher/driver roles feature). Kept fully idempotent (IF NOT
-- EXISTS everywhere) so it's a safe no-op against an already-populated dev
-- database, and a full bootstrap against a brand new one. Every migration
-- from 00002 onward is a normal, non-idempotent migration - goose itself now
-- guarantees each file only ever runs once.
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

-- fuel_percent/next_service_km are manually set by the driver, not sensor-derived -
-- this project has no telematics integration.
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS fuel_percent DOUBLE PRECISION NOT NULL DEFAULT 100;
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS next_service_km DOUBLE PRECISION;

ALTER TABLE trips ADD COLUMN IF NOT EXISTS cargo_description TEXT;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS cargo_weight_kg DOUBLE PRECISION;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS cargo_temp_range TEXT;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS pickup_location TEXT;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS dropoff_location TEXT;

CREATE TABLE IF NOT EXISTS trip_events (
    id SERIAL PRIMARY KEY,
    trip_id INTEGER NOT NULL REFERENCES trips(id),
    event_type TEXT NOT NULL,
    description TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id SERIAL PRIMARY KEY,
    from_driver_id INTEGER NOT NULL REFERENCES drivers(id),
    to_driver_id INTEGER NOT NULL REFERENCES drivers(id),
    body TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at TIMESTAMPTZ
);

-- +goose Down
-- No-op: this is the bootstrap baseline, tearing it down isn't useful and
-- wasn't requested.
SELECT 1;
