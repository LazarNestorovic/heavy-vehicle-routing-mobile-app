package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "correct-horse-battery-staple") {
		t.Error("expected correct password to check out")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Error("expected wrong password to fail")
	}
}

func TestIssueAndParseToken(t *testing.T) {
	m := New("test-secret")

	token, err := m.IssueToken(42, "lazar", 3)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	driverID, tokenVersion, err := m.ParseToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if driverID != 42 {
		t.Errorf("expected driver_id 42, got %d", driverID)
	}
	if tokenVersion != 3 {
		t.Errorf("expected token_version 3, got %d", tokenVersion)
	}
}

func TestParseToken_WrongSecretRejected(t *testing.T) {
	issuer := New("secret-a")
	verifier := New("secret-b")

	token, err := issuer.IssueToken(1, "lazar", 0)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, _, err := verifier.ParseToken(token); err == nil {
		t.Error("expected token signed with a different secret to be rejected")
	}
}

func TestParseToken_ExpiredRejected(t *testing.T) {
	m := New("test-secret")

	c := claims{
		DriverID: 1,
		Username: "lazar",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * tokenTTL)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // expired an hour ago
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, _, err := m.ParseToken(signed); err == nil {
		t.Error("expected expired token to be rejected")
	}
}

func TestParseToken_GarbageRejected(t *testing.T) {
	m := New("test-secret")
	if _, _, err := m.ParseToken("not-a-real-token"); err == nil {
		t.Error("expected garbage input to be rejected")
	}
}
