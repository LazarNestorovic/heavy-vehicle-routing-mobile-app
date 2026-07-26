package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// tokenTTL (24h) is shared with email_verification_token.go's constant of the
// same name in spirit, but kept separate since the two token types are
// otherwise unrelated and might reasonably diverge later.
const passwordResetTokenTTL = time.Hour

type PasswordResetToken struct {
	ID        int64
	DriverID  int64
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

type PasswordResetTokenStore struct {
	db *sql.DB
}

func NewPasswordResetTokenStore(db *sql.DB) *PasswordResetTokenStore {
	return &PasswordResetTokenStore{db: db}
}

// Create generates a new random token for driverID - shorter-lived than an
// email verification token (1h, not 24h) since a live reset link is a more
// sensitive thing to leave outstanding.
func (s *PasswordResetTokenStore) Create(ctx context.Context, driverID int64) (PasswordResetToken, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return PasswordResetToken{}, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(raw)

	t := PasswordResetToken{DriverID: driverID, Token: token, ExpiresAt: time.Now().Add(passwordResetTokenTTL)}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO password_reset_tokens (driver_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`, driverID, token, t.ExpiresAt)
	if err := row.Scan(&t.ID, &t.CreatedAt); err != nil {
		return PasswordResetToken{}, fmt.Errorf("insert reset token: %w", err)
	}
	return t, nil
}

// Consume looks up token, marks it used, and returns the driver_id it was
// for - same one-shot pattern as EmailVerificationTokenStore.Consume.
func (s *PasswordResetTokenStore) Consume(ctx context.Context, token string) (int64, error) {
	var t PasswordResetToken
	row := s.db.QueryRowContext(ctx, `
		SELECT id, driver_id, expires_at, used_at FROM password_reset_tokens WHERE token = $1`, token)
	if err := row.Scan(&t.ID, &t.DriverID, &t.ExpiresAt, &t.UsedAt); err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("select reset token: %w", err)
	}
	if t.UsedAt != nil {
		return 0, ErrTokenUsed
	}
	if time.Now().After(t.ExpiresAt) {
		return 0, ErrTokenExpired
	}

	if _, err := s.db.ExecContext(ctx, `UPDATE password_reset_tokens SET used_at = now() WHERE id = $1`, t.ID); err != nil {
		return 0, fmt.Errorf("mark reset token used: %w", err)
	}
	return t.DriverID, nil
}
