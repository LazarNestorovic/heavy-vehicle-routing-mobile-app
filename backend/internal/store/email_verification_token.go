package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// tokenTTL is how long a verification link stays valid before the driver
// needs to request a new one (handleResendVerification).
const tokenTTL = 24 * time.Hour

type EmailVerificationToken struct {
	ID        int64
	DriverID  int64
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

type EmailVerificationTokenStore struct {
	db *sql.DB
}

func NewEmailVerificationTokenStore(db *sql.DB) *EmailVerificationTokenStore {
	return &EmailVerificationTokenStore{db: db}
}

// Create generates a new random token for driverID and stores it with a
// tokenTTL expiry. The token itself (not a database id) is what goes in the
// emailed link - see httpapi handleVerifyEmail.
func (s *EmailVerificationTokenStore) Create(ctx context.Context, driverID int64) (EmailVerificationToken, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return EmailVerificationToken{}, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(raw)

	t := EmailVerificationToken{DriverID: driverID, Token: token, ExpiresAt: time.Now().Add(tokenTTL)}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO email_verification_tokens (driver_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`, driverID, token, t.ExpiresAt)
	if err := row.Scan(&t.ID, &t.CreatedAt); err != nil {
		return EmailVerificationToken{}, fmt.Errorf("insert verification token: %w", err)
	}
	return t, nil
}

var (
	// ErrTokenExpired and ErrTokenUsed let handleVerifyEmail show a more
	// specific message than a generic "not found".
	ErrTokenExpired = fmt.Errorf("store: verification token expired")
	ErrTokenUsed    = fmt.Errorf("store: verification token already used")
)

// Consume looks up token, marks it used, and returns the driver_id it was
// for - all in one call, since a verification link is only ever meant to be
// followed once.
func (s *EmailVerificationTokenStore) Consume(ctx context.Context, token string) (int64, error) {
	var t EmailVerificationToken
	row := s.db.QueryRowContext(ctx, `
		SELECT id, driver_id, expires_at, used_at FROM email_verification_tokens WHERE token = $1`, token)
	if err := row.Scan(&t.ID, &t.DriverID, &t.ExpiresAt, &t.UsedAt); err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("select verification token: %w", err)
	}
	if t.UsedAt != nil {
		return 0, ErrTokenUsed
	}
	if time.Now().After(t.ExpiresAt) {
		return 0, ErrTokenExpired
	}

	if _, err := s.db.ExecContext(ctx, `UPDATE email_verification_tokens SET used_at = now() WHERE id = $1`, t.ID); err != nil {
		return 0, fmt.Errorf("mark verification token used: %w", err)
	}
	return t.DriverID, nil
}
