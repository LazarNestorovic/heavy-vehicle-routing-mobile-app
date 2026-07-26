-- +goose Up
-- Google-only accounts have no password - password_hash was NOT NULL since
-- 00001, drop that so Create can leave it NULL for those.
ALTER TABLE drivers ALTER COLUMN password_hash DROP NOT NULL;

-- google_sub is Google's stable per-account identifier (the ID token's `sub`
-- claim) - the primary lookup key for "does this Google account already have
-- a driver row". email is a secondary lookup key used to link a Google
-- sign-in to a pre-existing username/password account that used the same
-- address (see documentations/features/ entry).
ALTER TABLE drivers ADD COLUMN google_sub TEXT UNIQUE;
ALTER TABLE drivers ADD COLUMN email TEXT UNIQUE;
ALTER TABLE drivers ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT false;

-- One row per outstanding (or spent) verification link sent to a driver's
-- email. expires_at/used_at instead of deleting on use, so a reused/expired
-- link can be told apart from one that never existed (better error message,
-- and a paper trail).
CREATE TABLE email_verification_tokens (
    id SERIAL PRIMARY KEY,
    driver_id INTEGER NOT NULL REFERENCES drivers(id),
    token TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE email_verification_tokens;
ALTER TABLE drivers DROP COLUMN email_verified;
ALTER TABLE drivers DROP COLUMN email;
ALTER TABLE drivers DROP COLUMN google_sub;
ALTER TABLE drivers ALTER COLUMN password_hash SET NOT NULL;
