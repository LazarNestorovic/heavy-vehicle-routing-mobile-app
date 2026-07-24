package httpapi

import (
	"encoding/json"
	"net/http"

	"heavy-vehicle-routing/backend/internal/auth"
	"heavy-vehicle-routing/backend/internal/store"
)

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token    string `json:"token"`
	DriverID int64  `json:"driver_id"`
}

func (req credentialsRequest) validate() error {
	if len(req.Username) < 3 {
		return errValidation("username must be at least 3 characters")
	}
	if len(req.Password) < 6 {
		return errValidation("password must be at least 6 characters")
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

	driver, err := s.Drivers.Create(r.Context(), req.Username, hash)
	if err != nil {
		if err == store.ErrDuplicateUsername {
			writeError(w, http.StatusConflict, "username already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create driver: "+err.Error())
		return
	}

	token, err := s.Auth.IssueToken(driver.ID, driver.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{Token: token, DriverID: driver.ID})
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
	if !auth.CheckPassword(driver.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := s.Auth.IssueToken(driver.ID, driver.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, DriverID: driver.ID})
}
