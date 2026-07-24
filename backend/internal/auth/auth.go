// Package auth handles driver password hashing and JWT issuing/verification.
// Tokens are stateless (HS256-signed JWTs, no server-side session table) - see
// documentations/features/... for the accepted trade-off (no server-side revocation,
// "logout" is just discarding the token client-side).
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 30 * 24 * time.Hour

var ErrInvalidToken = errors.New("auth: invalid or expired token")

type Manager struct {
	secret []byte
}

func New(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type claims struct {
	DriverID int64  `json:"driver_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (m *Manager) IssueToken(driverID int64, username string) (string, error) {
	now := time.Now()
	c := claims{
		DriverID: driverID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(m.secret)
}

// ParseToken verifies the signature and expiry, returning the driver_id claim.
func (m *Manager) ParseToken(tokenString string) (int64, error) {
	var c claims
	token, err := jwt.ParseWithClaims(tokenString, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return 0, ErrInvalidToken
	}
	return c.DriverID, nil
}
