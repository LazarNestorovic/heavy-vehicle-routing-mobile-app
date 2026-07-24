package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Driver struct {
	ID           int64
	Username     string
	PasswordHash string
}

var ErrDuplicateUsername = errors.New("store: username already taken")

type DriverStore struct {
	db *sql.DB
}

func NewDriverStore(db *sql.DB) *DriverStore {
	return &DriverStore{db: db}
}

func (s *DriverStore) Create(ctx context.Context, username, passwordHash string) (Driver, error) {
	d := Driver{Username: username, PasswordHash: passwordHash}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO drivers (username, password_hash) VALUES ($1, $2)
		RETURNING id`, username, passwordHash)

	if err := row.Scan(&d.ID); err != nil {
		if isUniqueViolation(err) {
			return Driver{}, ErrDuplicateUsername
		}
		return Driver{}, fmt.Errorf("insert driver: %w", err)
	}
	return d, nil
}

func (s *DriverStore) GetByUsername(ctx context.Context, username string) (Driver, error) {
	var d Driver
	d.Username = username
	row := s.db.QueryRowContext(ctx, `
		SELECT id, password_hash FROM drivers WHERE username = $1`, username)

	if err := row.Scan(&d.ID, &d.PasswordHash); err != nil {
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
