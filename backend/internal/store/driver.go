package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// RoleDriver and RoleDispatcher are the only valid values of Driver.Role (see
// the CHECK constraint in migrations/00002_dispatcher_roles.sql). Despite the
// name, "drivers" is the account table for both roles - see documentations/
// features/ entry for why this wasn't split into a separate table.
const (
	RoleDriver     = "driver"
	RoleDispatcher = "dispatcher"
)

// Driver.PasswordHash is nullable - a Google-only account (see CreateGoogle)
// never sets one. GoogleSub/Email/EmailVerified are all optional too, since
// a username/password account may have neither until it registers with an
// email or links a Google sign-in (see LinkGoogleSub).
type Driver struct {
	ID            int64
	Username      string
	PasswordHash  *string
	Role          string
	DispatcherID  *int64
	GoogleSub     *string
	Email         *string
	EmailVerified bool
	// TokenVersion is embedded in every issued JWT and checked on every
	// authenticated request (RequireAuth) - incrementing it (see
	// IncrementTokenVersion) invalidates every previously issued token for
	// this account without a server-side blocklist table.
	TokenVersion int
}

var (
	ErrDuplicateUsername = errors.New("store: username already taken")
	ErrDuplicateEmail    = errors.New("store: email already registered")
)

type DriverStore struct {
	db *sql.DB
}

func NewDriverStore(db *sql.DB) *DriverStore {
	return &DriverStore{db: db}
}

const driverColumns = "id, username, password_hash, role, dispatcher_id, google_sub, email, email_verified, token_version"

func scanDriver(row interface{ Scan(...any) error }, d *Driver) error {
	return row.Scan(&d.ID, &d.Username, &d.PasswordHash, &d.Role, &d.DispatcherID, &d.GoogleSub, &d.Email, &d.EmailVerified, &d.TokenVersion)
}

// Create registers a new username/password account with the given role and
// optional email (see documentations/features/ entry - email is opt-in on
// registration, verified separately via internal/mailer). The dispatcher<->
// driver relationship is never set here - see DispatcherRequestStore.
func (s *DriverStore) Create(ctx context.Context, username, passwordHash, role string, email *string) (Driver, error) {
	d := Driver{Username: username, PasswordHash: &passwordHash, Role: role, Email: email}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO drivers (username, password_hash, role, email) VALUES ($1, $2, $3, $4)
		RETURNING id`, username, passwordHash, role, email)

	if err := row.Scan(&d.ID); err != nil {
		if isUniqueViolation(err) {
			if strings.Contains(err.Error(), "email") {
				return Driver{}, ErrDuplicateEmail
			}
			return Driver{}, ErrDuplicateUsername
		}
		return Driver{}, fmt.Errorf("insert driver: %w", err)
	}
	return d, nil
}

// CreateGoogle registers a new account from a verified Google sign-in - no
// password, email_verified is trusted straight from Google's own claim (see
// internal/auth.GoogleClaims).
func (s *DriverStore) CreateGoogle(ctx context.Context, username, googleSub, role string, email *string, emailVerified bool) (Driver, error) {
	d := Driver{Username: username, Role: role, GoogleSub: &googleSub, Email: email, EmailVerified: emailVerified}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO drivers (username, role, google_sub, email, email_verified) VALUES ($1, $2, $3, $4, $5)
		RETURNING id`, username, role, googleSub, email, emailVerified)

	if err := row.Scan(&d.ID); err != nil {
		if isUniqueViolation(err) {
			return Driver{}, ErrDuplicateUsername
		}
		return Driver{}, fmt.Errorf("insert google driver: %w", err)
	}
	return d, nil
}

// Get loads an account by id - used to check the caller's role/dispatcher_id
// for role-aware handlers (the JWT only carries driver_id/username, see
// internal/auth, so this is a deliberate per-request lookup rather than
// caching role in the token).
func (s *DriverStore) Get(ctx context.Context, id int64) (Driver, error) {
	d := Driver{ID: id}
	row := s.db.QueryRowContext(ctx, `SELECT `+driverColumns+` FROM drivers WHERE id = $1`, id)
	if err := scanDriver(row, &d); err != nil {
		if err == sql.ErrNoRows {
			return Driver{}, ErrNotFound
		}
		return Driver{}, fmt.Errorf("select driver: %w", err)
	}
	return d, nil
}

// GetByGoogleSub looks up an account previously created/linked via Google
// sign-in.
func (s *DriverStore) GetByGoogleSub(ctx context.Context, googleSub string) (Driver, error) {
	var d Driver
	row := s.db.QueryRowContext(ctx, `SELECT `+driverColumns+` FROM drivers WHERE google_sub = $1`, googleSub)
	if err := scanDriver(row, &d); err != nil {
		if err == sql.ErrNoRows {
			return Driver{}, ErrNotFound
		}
		return Driver{}, fmt.Errorf("select driver by google sub: %w", err)
	}
	return d, nil
}

// GetByEmail looks up an account by email - used to link a Google sign-in to
// a pre-existing username/password account that used the same address.
func (s *DriverStore) GetByEmail(ctx context.Context, email string) (Driver, error) {
	var d Driver
	row := s.db.QueryRowContext(ctx, `SELECT `+driverColumns+` FROM drivers WHERE email = $1`, email)
	if err := scanDriver(row, &d); err != nil {
		if err == sql.ErrNoRows {
			return Driver{}, ErrNotFound
		}
		return Driver{}, fmt.Errorf("select driver by email: %w", err)
	}
	return d, nil
}

// LinkGoogleSub attaches a Google account to a pre-existing driver row (found
// via GetByEmail) - so future sign-ins with that Google account resolve
// straight to google_sub instead of relying on the email match again.
func (s *DriverStore) LinkGoogleSub(ctx context.Context, driverID int64, googleSub string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE drivers SET google_sub = $2 WHERE id = $1`, driverID, googleSub)
	if err != nil {
		return fmt.Errorf("link google sub: %w", err)
	}
	return nil
}

// MarkEmailVerified sets email_verified=true - called once a verification
// token is successfully consumed (see EmailVerificationTokenStore.Consume).
func (s *DriverStore) MarkEmailVerified(ctx context.Context, driverID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE drivers SET email_verified = true WHERE id = $1`, driverID)
	if err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	return nil
}

// IncrementTokenVersion bumps token_version, invalidating every JWT issued
// before this call (see Driver.TokenVersion) - the "logout everywhere"
// primitive, called from POST /api/v1/auth/logout-all.
func (s *DriverStore) IncrementTokenVersion(ctx context.Context, driverID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE drivers SET token_version = token_version + 1 WHERE id = $1`, driverID)
	if err != nil {
		return fmt.Errorf("increment token version: %w", err)
	}
	return nil
}

// SetPasswordHash overwrites a driver's password hash - used by the
// forgot-password flow (handleResetPassword) once a reset token has been
// consumed. Also bumps token_version, so a password reset (e.g. because the
// old one leaked) invalidates any tokens issued under the old password too.
func (s *DriverStore) SetPasswordHash(ctx context.Context, driverID int64, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE drivers SET password_hash = $2, token_version = token_version + 1 WHERE id = $1`, driverID, passwordHash)
	if err != nil {
		return fmt.Errorf("set password hash: %w", err)
	}
	return nil
}

// SetDispatcher links driverID to dispatcherID - called only from an approved
// DispatcherRequest, never directly.
func (s *DriverStore) SetDispatcher(ctx context.Context, driverID, dispatcherID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE drivers SET dispatcher_id = $2 WHERE id = $1`, driverID, dispatcherID)
	if err != nil {
		return fmt.Errorf("set dispatcher: %w", err)
	}
	return nil
}

// ClearDispatcher removes driverID's dispatcher link - the driver-initiated
// counterpart to SetDispatcher (see handleLeaveDispatcher). Fleet vehicles
// and any already-created trips are untouched: vehicleAccessible/
// tripAccessible check the CURRENT dispatcher_id, so access to the former
// dispatcher's fleet simply ends naturally, while past trips (keyed by
// driver_id/assigned_by_id, not the live dispatcher_id) remain in history.
func (s *DriverStore) ClearDispatcher(ctx context.Context, driverID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE drivers SET dispatcher_id = NULL WHERE id = $1`, driverID)
	if err != nil {
		return fmt.Errorf("clear dispatcher: %w", err)
	}
	return nil
}

// ListManaged returns every driver managed by dispatcherID.
func (s *DriverStore) ListManaged(ctx context.Context, dispatcherID int64) ([]Driver, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username FROM drivers WHERE dispatcher_id = $1 ORDER BY username ASC`, dispatcherID)
	if err != nil {
		return nil, fmt.Errorf("list managed drivers: %w", err)
	}
	defer rows.Close()

	drivers := []Driver{}
	for rows.Next() {
		d := Driver{DispatcherID: &dispatcherID}
		if err := rows.Scan(&d.ID, &d.Username); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		drivers = append(drivers, d)
	}
	return drivers, rows.Err()
}

// ListAvailable returns every unmanaged driver (role='driver', no dispatcher
// yet) whose username or email contains query as a case-insensitive
// substring - an empty query matches everyone. This is the dispatcher's
// "send a request" contact list, now searchable (see documentations/
// features/ entry).
func (s *DriverStore) ListAvailable(ctx context.Context, query string) ([]Driver, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, email FROM drivers
		WHERE role = $1 AND dispatcher_id IS NULL
		  AND (LOWER(username) LIKE $2 OR LOWER(COALESCE(email, '')) LIKE $2)
		ORDER BY username ASC`, RoleDriver, pattern)
	if err != nil {
		return nil, fmt.Errorf("list available drivers: %w", err)
	}
	defer rows.Close()

	drivers := []Driver{}
	for rows.Next() {
		var d Driver
		if err := rows.Scan(&d.ID, &d.Username, &d.Email); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		drivers = append(drivers, d)
	}
	return drivers, rows.Err()
}

// List returns every registered account except excludeID (the caller - used
// to populate the "start a new chat" contact list) whose username or email
// contains query as a case-insensitive substring - an empty query matches
// everyone. No fleet/team concept yet (see documentations/features/2026-07-21-
// nocturne-redesign.md) - every registered account (driver or dispatcher) is
// a valid chat contact; pinning the caller's own dispatcher/managed drivers
// to the top of the list is done client-side (see documentations/features/
// entry for the chat contact search + pinning feature).
func (s *DriverStore) List(ctx context.Context, excludeID int64, query string) ([]Driver, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, email FROM drivers
		WHERE id != $1 AND (LOWER(username) LIKE $2 OR LOWER(COALESCE(email, '')) LIKE $2)
		ORDER BY username ASC`, excludeID, pattern)
	if err != nil {
		return nil, fmt.Errorf("list drivers: %w", err)
	}
	defer rows.Close()

	drivers := []Driver{}
	for rows.Next() {
		var d Driver
		if err := rows.Scan(&d.ID, &d.Username, &d.Email); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		drivers = append(drivers, d)
	}
	return drivers, rows.Err()
}

func (s *DriverStore) GetByUsername(ctx context.Context, username string) (Driver, error) {
	var d Driver
	row := s.db.QueryRowContext(ctx, `SELECT `+driverColumns+` FROM drivers WHERE username = $1`, username)
	if err := scanDriver(row, &d); err != nil {
		if err == sql.ErrNoRows {
			return Driver{}, ErrNotFound
		}
		return Driver{}, fmt.Errorf("select driver: %w", err)
	}
	return d, nil
}

// isUniqueViolation checks for Postgres SQLSTATE 23505 without importing the
// pgx error types directly - a plain substring check on the driver error is
// enough for our one use case here.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}
