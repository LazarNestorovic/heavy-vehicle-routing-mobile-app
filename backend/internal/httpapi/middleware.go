package httpapi

import (
	"context"
	"net/http"
	"strings"
)

type contextKey int

const driverIDContextKey contextKey = iota

// RequireAuth wraps a handler so it only runs for requests carrying a valid
// "Authorization: Bearer <jwt>" header, and makes the driver_id claim
// available to the handler via driverIDFromContext.
func (s *Server) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			writeError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		driverID, tokenVersion, err := s.Auth.ParseToken(strings.TrimPrefix(header, prefix))
		if err != nil || !s.tokenVersionValid(r.Context(), driverID, tokenVersion) {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), driverIDContextKey, driverID)))
	}
}

// RequireAuthQuery is RequireAuth's counterpart for the WebSocket endpoint:
// browsers' WebSocket API can't set custom headers on the handshake, so the
// token travels as a query parameter (?token=...) instead.
func (s *Server) RequireAuthQuery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		driverID, tokenVersion, err := s.Auth.ParseToken(r.URL.Query().Get("token"))
		if err != nil || !s.tokenVersionValid(r.Context(), driverID, tokenVersion) {
			writeError(w, http.StatusUnauthorized, "missing or invalid token query parameter")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), driverIDContextKey, driverID)))
	}
}

// tokenVersionValid checks the claim against the database's current value -
// one extra query per authenticated request, an accepted cost at this
// project's scale (see documentations/features/ live-GPS/auth entry) in
// exchange for real "logout everywhere" support without a blocklist table.
func (s *Server) tokenVersionValid(ctx context.Context, driverID int64, tokenVersion int) bool {
	d, err := s.Drivers.Get(ctx, driverID)
	if err != nil {
		return false
	}
	return d.TokenVersion == tokenVersion
}

func driverIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(driverIDContextKey).(int64)
	return id, ok
}
