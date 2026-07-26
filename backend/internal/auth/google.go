package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// googleJWKSURL serves Google's current signing keys for ID tokens - see
// documentations/guides/google-maps-setup.md for the OAuth client setup this
// depends on. keyfunc fetches and background-refreshes this automatically
// rather than us hand-rolling JWKS caching/RSA-from-JWK parsing (deliberately
// small, focused dependency - see documentations/features/ entry).
const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

var ErrInvalidGoogleToken = errors.New("auth: invalid google id token")

// GoogleClaims is what we trust out of a verified Google ID token.
// EmailVerified comes straight from Google, which already confirmed it -
// see httpapi handleGoogleAuth for why that's enough to skip our own
// verification email for Google-created accounts.
type GoogleClaims struct {
	Sub           string
	Email         string
	EmailVerified bool
	Name          string
}

type googleIDTokenClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	jwt.RegisteredClaims
}

// GoogleVerifier holds the (self-refreshing) set of Google's public signing
// keys, plus our own OAuth client ID for audience validation.
type GoogleVerifier struct {
	clientID string
	keyfunc  keyfunc.Keyfunc
}

// NewGoogleVerifier fetches Google's JWKS once at startup (keyfunc refreshes
// it in the background afterward). clientID is the Web OAuth client ID from
// documentations/guides/google-maps-setup.md step 7 - it must match the
// `aud` claim on every token we verify.
func NewGoogleVerifier(ctx context.Context, clientID string) (*GoogleVerifier, error) {
	k, err := keyfunc.NewDefaultCtx(ctx, []string{googleJWKSURL})
	if err != nil {
		return nil, fmt.Errorf("fetch google jwks: %w", err)
	}
	return &GoogleVerifier{clientID: clientID, keyfunc: k}, nil
}

// VerifyIDToken checks the signature (against Google's current keys),
// issuer, audience (our client ID), and expiry (built into jwt.ParseWithClaims),
// returning the claims we care about.
func (v *GoogleVerifier) VerifyIDToken(idToken string) (GoogleClaims, error) {
	var claims googleIDTokenClaims
	token, err := jwt.ParseWithClaims(idToken, &claims, v.keyfunc.Keyfunc)
	if err != nil || !token.Valid {
		return GoogleClaims{}, ErrInvalidGoogleToken
	}

	if claims.Issuer != "accounts.google.com" && claims.Issuer != "https://accounts.google.com" {
		return GoogleClaims{}, ErrInvalidGoogleToken
	}
	validAudience := false
	for _, aud := range claims.Audience {
		if aud == v.clientID {
			validAudience = true
			break
		}
	}
	if !validAudience || claims.Subject == "" {
		return GoogleClaims{}, ErrInvalidGoogleToken
	}

	return GoogleClaims{
		Sub:           claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
	}, nil
}
