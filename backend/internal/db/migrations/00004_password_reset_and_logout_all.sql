-- +goose Up
-- token_version is embedded in every issued JWT (auth.claims) and checked on
-- every authenticated request (RequireAuth) - incrementing it (logout-all)
-- immediately invalidates every previously issued token for that driver,
-- without a server-side blocklist table.
ALTER TABLE drivers ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;

-- Same shape as email_verification_tokens (see 00003) - one row per
-- outstanding (or spent) reset link, expires_at/used_at instead of deleting
-- on use so a reused/expired link can be told apart from one that never
-- existed.
CREATE TABLE password_reset_tokens (
    id SERIAL PRIMARY KEY,
    driver_id INTEGER NOT NULL REFERENCES drivers(id),
    token TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE password_reset_tokens;
ALTER TABLE drivers DROP COLUMN token_version;
