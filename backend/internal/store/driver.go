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

type Driver struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	DispatcherID *int64
}

var ErrDuplicateUsername = errors.New("store: username already taken")

type DriverStore struct {
	db *sql.DB
}

func NewDriverStore(db *sql.DB) *DriverStore {
	return &DriverStore{db: db}
}

// Create registers a new account with the given role. The dispatcher<->driver
// relationship is never set here - see DispatcherRequestStore.
func (s *DriverStore) Create(ctx context.Context, username, passwordHash, role string) (Driver, error) {
	d := Driver{Username: username, PasswordHash: passwordHash, Role: role}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO drivers (username, password_hash, role) VALUES ($1, $2, $3)
		RETURNING id`, username, passwordHash, role)

	if err := row.Scan(&d.ID); err != nil {
		if isUniqueViolation(err) {
			return Driver{}, ErrDuplicateUsername
		}
		return Driver{}, fmt.Errorf("insert driver: %w", err)
	}
	return d, nil
}

// Get loads an account by id - used to check the caller's role/dispatcher_id
// for role-aware handlers (the JWT only carries driver_id/username, see
// internal/auth, so this is a deliberate per-request lookup rather than
// caching role in the token).
func (s *DriverStore) Get(ctx context.Context, id int64) (Driver, error) {
	d := Driver{ID: id}
	row := s.db.QueryRowContext(ctx, `
		SELECT username, password_hash, role, dispatcher_id FROM drivers WHERE id = $1`, id)

	if err := row.Scan(&d.Username, &d.PasswordHash, &d.Role, &d.DispatcherID); err != nil {
		if err == sql.ErrNoRows {
			return Driver{}, ErrNotFound
		}
		return Driver{}, fmt.Errorf("select driver: %w", err)
	}
	return d, nil
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
// yet) - the dispatcher's "send a request" contact list.
func (s *DriverStore) ListAvailable(ctx context.Context) ([]Driver, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username FROM drivers WHERE role = $1 AND dispatcher_id IS NULL ORDER BY username ASC`, RoleDriver)
	if err != nil {
		return nil, fmt.Errorf("list available drivers: %w", err)
	}
	defer rows.Close()

	drivers := []Driver{}
	for rows.Next() {
		var d Driver
		if err := rows.Scan(&d.ID, &d.Username); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		drivers = append(drivers, d)
	}
	return drivers, rows.Err()
}

// List returns every registered driver except excludeID (the caller - used to
// populate the "start a new chat" contact list). No fleet/team concept yet
// (see documentations/features/2026-07-21-nocturne-redesign.md) - every
// registered driver is a valid chat contact.
func (s *DriverStore) List(ctx context.Context, excludeID int64) ([]Driver, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username FROM drivers WHERE id != $1 ORDER BY username ASC`, excludeID)
	if err != nil {
		return nil, fmt.Errorf("list drivers: %w", err)
	}
	defer rows.Close()

	drivers := []Driver{}
	for rows.Next() {
		var d Driver
		if err := rows.Scan(&d.ID, &d.Username); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		drivers = append(drivers, d)
	}
	return drivers, rows.Err()
}

func (s *DriverStore) GetByUsername(ctx context.Context, username string) (Driver, error) {
	var d Driver
	d.Username = username
	row := s.db.QueryRowContext(ctx, `
		SELECT id, password_hash, role, dispatcher_id FROM drivers WHERE username = $1`, username)

	if err := row.Scan(&d.ID, &d.PasswordHash, &d.Role, &d.DispatcherID); err != nil {
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
