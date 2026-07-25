-- +goose Up
ALTER TABLE drivers ADD COLUMN role TEXT NOT NULL DEFAULT 'driver' CHECK (role IN ('driver', 'dispatcher'));
ALTER TABLE drivers ADD COLUMN dispatcher_id INTEGER REFERENCES drivers(id);

-- vehicles.driver_id is already nullable (00001) - exactly one of
-- driver_id/dispatcher_id should be set per vehicle. That check stays at the
-- Go handler level, not a DB CHECK (see documentations/features/ entry).
ALTER TABLE vehicles ADD COLUMN dispatcher_id INTEGER REFERENCES drivers(id);

-- trips.driver_id remains the ASSIGNED driver (as before) - all existing
-- ownership checks, WS auth, chat, and trip_events code works unchanged for
-- dispatcher-assigned trips too. assigned_by_id is NULL for self-service
-- trips, set to the dispatcher for assigned ones. status gains a new value
-- 'offered' (before 'created') for trips awaiting the driver's start.
ALTER TABLE trips ADD COLUMN assigned_by_id INTEGER REFERENCES drivers(id);

-- The dispatcher<->driver relationship is established exclusively through
-- this table (request + approval), never at registration. Approval sets
-- drivers.dispatcher_id.
CREATE TABLE dispatcher_requests (
    id SERIAL PRIMARY KEY,
    dispatcher_id INTEGER NOT NULL REFERENCES drivers(id),
    driver_id INTEGER NOT NULL REFERENCES drivers(id),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE dispatcher_requests;
ALTER TABLE trips DROP COLUMN assigned_by_id;
ALTER TABLE vehicles DROP COLUMN dispatcher_id;
ALTER TABLE drivers DROP COLUMN dispatcher_id;
ALTER TABLE drivers DROP COLUMN role;
