package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"heavy-vehicle-routing/backend/internal/auth"
	"heavy-vehicle-routing/backend/internal/store"
)

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// Role is optional at registration ("driver"/"dispatcher", default
	// "driver"); ignored on login. The dispatcher<->driver link is never set
	// here - see dispatcher_requests / handleRespondDispatcherRequest.
	Role string `json:"role,omitempty"`
	// Email is required at registration (see validate()) - a verification
	// link is always sent (see internal/mailer and handleVerifyEmail).
	Email *string `json:"email,omitempty"`
}

type authResponse struct {
	Token         string  `json:"token"`
	DriverID      int64   `json:"driver_id"`
	Username      string  `json:"username"`
	Role          string  `json:"role"`
	DispatcherID  *int64  `json:"dispatcher_id,omitempty"`
	Email         *string `json:"email,omitempty"`
	EmailVerified bool    `json:"email_verified"`
}

func toAuthResponse(token string, d store.Driver) authResponse {
	return authResponse{
		Token: token, DriverID: d.ID, Username: d.Username, Role: d.Role, DispatcherID: d.DispatcherID,
		Email: d.Email, EmailVerified: d.EmailVerified,
	}
}

func (req credentialsRequest) validate() error {
	if len(req.Username) < 3 {
		return errValidation("username must be at least 3 characters")
	}
	if len(req.Password) < 6 {
		return errValidation("password must be at least 6 characters")
	}
	if req.Role != "" && req.Role != store.RoleDriver && req.Role != store.RoleDispatcher {
		return errValidation("role must be 'driver' or 'dispatcher'")
	}
	if req.Email == nil || !strings.Contains(*req.Email, "@") {
		return errValidation("email is required and must be valid")
	}
	return nil
}

type errValidation string

func (e errValidation) Error() string { return string(e) }

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password: "+err.Error())
		return
	}

	role := req.Role
	if role == "" {
		role = store.RoleDriver
	}

	driver, err := s.Drivers.Create(r.Context(), req.Username, hash, role, req.Email)
	if err != nil {
		switch err {
		case store.ErrDuplicateUsername:
			writeError(w, http.StatusConflict, "username already taken")
		case store.ErrDuplicateEmail:
			writeError(w, http.StatusConflict, "email already registered")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create driver: "+err.Error())
		}
		return
	}

	token, err := s.Auth.IssueToken(driver.ID, driver.Username, driver.TokenVersion)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token: "+err.Error())
		return
	}

	s.sendVerificationEmailIfNeeded(r.Context(), driver)

	writeJSON(w, http.StatusCreated, toAuthResponse(token, driver))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	driver, err := s.Drivers.GetByUsername(r.Context(), req.Username)
	if err != nil {
		// Same generic message whether the username doesn't exist or the password
		// is wrong - don't leak which one it was.
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	// A Google-only account (see handleGoogleAuth) has no password_hash - reject
	// it the same as a wrong password rather than crashing on a nil deref.
	if driver.PasswordHash == nil || !auth.CheckPassword(*driver.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := s.Auth.IssueToken(driver.ID, driver.Username, driver.TokenVersion)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(token, driver))
}

type googleAuthRequest struct {
	IDToken string `json:"id_token"`
	// Role is only used when this Google account has no matching driver row
	// yet (first-time sign-in creates one); ignored for an existing account.
	Role string `json:"role,omitempty"`
}

// handleGoogleAuth verifies a Google ID token (see internal/auth.GoogleVerifier)
// and either logs into an existing account or creates a new one:
//  1. google_sub matches an existing driver -> log them in.
//  2. No google_sub match, but email matches a pre-existing username/password
//     account -> link this Google account to it (LinkGoogleSub) and log in.
//  3. Neither matches -> create a new account (role from the request,
//     email_verified trusted directly from Google's own claim).
func (s *Server) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	if s.GoogleAuth == nil {
		writeError(w, http.StatusServiceUnavailable, "google sign-in is not configured on this server")
		return
	}

	var req googleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Role != "" && req.Role != store.RoleDriver && req.Role != store.RoleDispatcher {
		writeError(w, http.StatusBadRequest, "role must be 'driver' or 'dispatcher'")
		return
	}

	claims, err := s.GoogleAuth.VerifyIDToken(req.IDToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid google id token")
		return
	}

	driver, err := s.Drivers.GetByGoogleSub(r.Context(), claims.Sub)
	if err != nil && err != store.ErrNotFound {
		writeError(w, http.StatusInternalServerError, "failed to load driver: "+err.Error())
		return
	}

	if err == store.ErrNotFound && claims.Email != "" {
		if existing, emailErr := s.Drivers.GetByEmail(r.Context(), claims.Email); emailErr == nil {
			if linkErr := s.Drivers.LinkGoogleSub(r.Context(), existing.ID, claims.Sub); linkErr != nil {
				writeError(w, http.StatusInternalServerError, "failed to link google account: "+linkErr.Error())
				return
			}
			existing.GoogleSub = &claims.Sub
			driver = existing
			err = nil
		} else if emailErr != store.ErrNotFound {
			writeError(w, http.StatusInternalServerError, "failed to load driver: "+emailErr.Error())
			return
		}
	}

	if err == store.ErrNotFound {
		role := req.Role
		if role == "" {
			role = store.RoleDriver
		}
		username := claims.Email
		if username == "" {
			username = "google_" + claims.Sub
		}
		var email *string
		if claims.Email != "" {
			email = &claims.Email
		}
		driver, err = s.Drivers.CreateGoogle(r.Context(), username, claims.Sub, role, email, claims.EmailVerified)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create driver: "+err.Error())
			return
		}
	}

	token, err := s.Auth.IssueToken(driver.ID, driver.Username, driver.TokenVersion)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(token, driver))
}

// sendVerificationEmailIfNeeded is a no-op unless the account has an email
// that isn't verified yet - a Google-created account is already verified
// (see handleGoogleAuth) and never reaches here. A mail failure is logged,
// not returned as an error - the account is already created either way, same
// reasoning as the trip.started queue publish elsewhere in this codebase.
func (s *Server) sendVerificationEmailIfNeeded(ctx context.Context, d store.Driver) {
	if d.Email == nil || d.EmailVerified {
		return
	}
	token, err := s.EmailVerifications.Create(ctx, d.ID)
	if err != nil {
		log.Printf("create verification token for driver %d: %v", d.ID, err)
		return
	}
	link := fmt.Sprintf("%s/api/v1/auth/verify-email?token=%s", s.PublicBackendURL, token.Token)
	body := fmt.Sprintf("Zdravo,\n\nPotvrdi svoju email adresu klikom na link ispod:\n%s\n\nLink važi 24 sata.", link)
	if err := s.Mailer.Send(*d.Email, "Potvrdi email adresu - Heavy Vehicle Routing", body); err != nil {
		log.Printf("send verification email to driver %d: %v", d.ID, err)
	}
}

// handleVerifyEmail is deliberately public (no JWT) - the token itself, sent
// only to the driver's own inbox, is the proof of identity. Clicking the
// emailed link just opens this in a browser; renders a plain HTML page
// rather than redirecting into the app (see documentations/features/ entry
// for why deep-linking was cut from scope).
func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	driverID, err := s.EmailVerifications.Consume(r.Context(), token)
	if err != nil {
		message := "Nevažeći link."
		switch err {
		case store.ErrTokenExpired:
			message = "Link je istekao. Zatraži novi iz aplikacije."
		case store.ErrTokenUsed:
			message = "Ovaj link je već iskorišćen."
		case store.ErrNotFound:
			message = "Nevažeći link."
		default:
			writeError(w, http.StatusInternalServerError, "failed to verify email: "+err.Error())
			return
		}
		writeHTML(w, http.StatusBadRequest, message)
		return
	}

	if err := s.Drivers.MarkEmailVerified(r.Context(), driverID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify email: "+err.Error())
		return
	}
	writeHTML(w, http.StatusOK, "Email adresa je potvrđena. Možeš se vratiti u aplikaciju.")
}

func writeHTML(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, "<!DOCTYPE html><html><body style=\"font-family:sans-serif;text-align:center;padding:48px\"><h2>%s</h2></body></html>", message)
}

type meResponse struct {
	DriverID      int64   `json:"driver_id"`
	Username      string  `json:"username"`
	Role          string  `json:"role"`
	DispatcherID  *int64  `json:"dispatcher_id,omitempty"`
	Email         *string `json:"email,omitempty"`
	EmailVerified bool    `json:"email_verified"`
}

// handleMe returns the caller's current account state - notably email_verified,
// which can change outside the app (the driver clicks the emailed link in a
// browser) with nothing pushing the update back to a running session. Lets
// the client refresh that (see EmailVerificationBanner) without a full
// re-login, which was previously the only way to pick it up.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	driver, err := s.Drivers.Get(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, meResponse{
		DriverID: driver.ID, Username: driver.Username, Role: driver.Role,
		DispatcherID: driver.DispatcherID, Email: driver.Email, EmailVerified: driver.EmailVerified,
	})
}

// handleResendVerification re-sends the verification email for the caller's
// own account.
func (s *Server) handleResendVerification(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	driver, err := s.Drivers.Get(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}
	if driver.Email == nil {
		writeError(w, http.StatusBadRequest, "no email on file")
		return
	}
	if driver.EmailVerified {
		writeError(w, http.StatusConflict, "email already verified")
		return
	}

	s.sendVerificationEmailIfNeeded(r.Context(), driver)
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// handleForgotPassword always returns the same generic response regardless
// of whether the email matches an account - doesn't leak which addresses are
// registered (standard account-enumeration defense). Public, no auth.
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if driver, err := s.Drivers.GetByEmail(r.Context(), req.Email); err == nil {
		s.sendPasswordResetEmail(r.Context(), driver)
	} else if err != store.ErrNotFound {
		log.Printf("forgot-password: load driver by email: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) sendPasswordResetEmail(ctx context.Context, d store.Driver) {
	token, err := s.PasswordResets.Create(ctx, d.ID)
	if err != nil {
		log.Printf("create reset token for driver %d: %v", d.ID, err)
		return
	}
	link := fmt.Sprintf("%s/api/v1/auth/reset-password?token=%s", s.PublicBackendURL, token.Token)
	body := fmt.Sprintf("Zdravo,\n\nZatražena je nova lozinka za tvoj nalog. Klikni link ispod da postaviš novu:\n%s\n\nLink važi 1 sat. Ako nisi ti tražio/la ovo, slobodno ignoriši ovaj mejl.", link)
	if err := s.Mailer.Send(*d.Email, "Resetovanje lozinke - Heavy Vehicle Routing", body); err != nil {
		log.Printf("send reset email to driver %d: %v", d.ID, err)
	}
}

// handleShowResetPasswordForm is the page the emailed link opens - a plain
// HTML form (same "browser, not deep-link" pattern as handleVerifyEmail)
// that posts the new password back to handleSubmitResetPassword.
func (s *Server) handleShowResetPasswordForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeHTML(w, http.StatusBadRequest, "Nevažeći link.")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html><html><body style="font-family:sans-serif;max-width:360px;margin:48px auto;padding:0 16px">
<h2>Nova lozinka</h2>
<form method="POST" action="/api/v1/auth/reset-password">
<input type="hidden" name="token" value="%s">
<input type="password" name="password" placeholder="Nova lozinka (min 6 karaktera)" minlength="6" required style="width:100%%;padding:8px;margin-bottom:12px;box-sizing:border-box">
<button type="submit" style="width:100%%;padding:8px">Postavi novu lozinku</button>
</form>
</body></html>`, token)
}

// handleSubmitResetPassword consumes the token and, if still valid, sets the
// new password hash - also bumping token_version (see DriverStore.
// SetPasswordHash), so any session started under the old password is
// invalidated too.
func (s *Server) handleSubmitResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeHTML(w, http.StatusBadRequest, "Nevažeći zahtev.")
		return
	}
	token := r.FormValue("token")
	password := r.FormValue("password")
	if len(password) < 6 {
		writeHTML(w, http.StatusBadRequest, "Lozinka mora imati bar 6 karaktera.")
		return
	}

	driverID, err := s.PasswordResets.Consume(r.Context(), token)
	if err != nil {
		message := "Nevažeći link."
		switch err {
		case store.ErrTokenExpired:
			message = "Link je istekao. Zatraži novi."
		case store.ErrTokenUsed:
			message = "Ovaj link je već iskorišćen."
		case store.ErrNotFound:
			message = "Nevažeći link."
		default:
			writeHTML(w, http.StatusInternalServerError, "Greška servera.")
			return
		}
		writeHTML(w, http.StatusBadRequest, message)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		writeHTML(w, http.StatusInternalServerError, "Greška servera.")
		return
	}
	if err := s.Drivers.SetPasswordHash(r.Context(), driverID, hash); err != nil {
		writeHTML(w, http.StatusInternalServerError, "Greška servera.")
		return
	}

	writeHTML(w, http.StatusOK, "Lozinka je promenjena. Možeš se vratiti u aplikaciju i ulogovati.")
}

// handleLogoutAll invalidates every token previously issued to the caller
// (see Driver.TokenVersion) - "logout everywhere" without a server-side
// blocklist table.
func (s *Server) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	if err := s.Drivers.IncrementTokenVersion(r.Context(), driverID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to log out: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}
