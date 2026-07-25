package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"heavy-vehicle-routing/backend/internal/store"
)

// requireDispatcher loads the caller's account and writes a 403 if they're
// not a dispatcher. Returns ok=false if a response was already written.
func (s *Server) requireDispatcher(w http.ResponseWriter, r *http.Request, callerID int64) (store.Driver, bool) {
	account, err := s.loadAccount(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return store.Driver{}, false
	}
	if account.Role != store.RoleDispatcher {
		writeError(w, http.StatusForbidden, "dispatcher role required")
		return store.Driver{}, false
	}
	return account, true
}

func (s *Server) handleListManagedDrivers(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())
	if _, ok := s.requireDispatcher(w, r, callerID); !ok {
		return
	}

	drivers, err := s.Drivers.ListManaged(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list managed drivers: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDriverResponses(drivers))
}

func (s *Server) handleListAvailableDrivers(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())
	if _, ok := s.requireDispatcher(w, r, callerID); !ok {
		return
	}

	drivers, err := s.Drivers.ListAvailable(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list available drivers: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDriverResponses(drivers))
}

func toDriverResponses(drivers []store.Driver) []driverResponse {
	out := make([]driverResponse, len(drivers))
	for i, d := range drivers {
		out[i] = driverResponse{ID: d.ID, Username: d.Username}
	}
	return out
}

type dispatcherRequestResponse struct {
	ID             int64  `json:"id"`
	DispatcherID   int64  `json:"dispatcher_id"`
	DispatcherName string `json:"dispatcher_username,omitempty"`
	DriverID       int64  `json:"driver_id"`
	DriverUsername string `json:"driver_username,omitempty"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

type createDispatcherRequestRequest struct {
	DriverID int64 `json:"driver_id"`
}

func (s *Server) handleCreateDispatcherRequest(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())
	if _, ok := s.requireDispatcher(w, r, callerID); !ok {
		return
	}

	var req createDispatcherRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	saved, err := s.DispatcherRequests.Create(r.Context(), callerID, req.DriverID)
	if err != nil {
		switch err {
		case store.ErrNotFound:
			writeError(w, http.StatusNotFound, "driver not found")
		case store.ErrAlreadyManaged:
			writeError(w, http.StatusConflict, "driver already has a dispatcher")
		case store.ErrRequestAlreadyPending:
			writeError(w, http.StatusConflict, "a pending request to this driver already exists")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create request: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, dispatcherRequestResponse{
		ID: saved.ID, DispatcherID: saved.DispatcherID, DriverID: saved.DriverID,
		Status: saved.Status, CreatedAt: saved.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleListDispatcherRequests(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())
	if _, ok := s.requireDispatcher(w, r, callerID); !ok {
		return
	}

	requests, err := s.DispatcherRequests.ListByDispatcher(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list requests: "+err.Error())
		return
	}

	out := make([]dispatcherRequestResponse, len(requests))
	for i, req := range requests {
		resp := dispatcherRequestResponse{
			ID: req.ID, DispatcherID: req.DispatcherID, DriverID: req.DriverID,
			Status: req.Status, CreatedAt: req.CreatedAt.Format(time.RFC3339),
		}
		if driver, err := s.Drivers.Get(r.Context(), req.DriverID); err == nil {
			resp.DriverUsername = driver.Username
		}
		out[i] = resp
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleListDriverRequests(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())

	requests, err := s.DispatcherRequests.ListPendingForDriver(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list requests: "+err.Error())
		return
	}

	out := make([]dispatcherRequestResponse, len(requests))
	for i, req := range requests {
		resp := dispatcherRequestResponse{
			ID: req.ID, DispatcherID: req.DispatcherID, DriverID: req.DriverID,
			Status: req.Status, CreatedAt: req.CreatedAt.Format(time.RFC3339),
		}
		if dispatcher, err := s.Drivers.Get(r.Context(), req.DispatcherID); err == nil {
			resp.DispatcherName = dispatcher.Username
		}
		out[i] = resp
	}
	writeJSON(w, http.StatusOK, out)
}

type respondDispatcherRequestRequest struct {
	Approve bool `json:"approve"`
}

type respondDispatcherRequestResponse struct {
	Status       string `json:"status"`
	DispatcherID *int64 `json:"dispatcher_id,omitempty"`
}

func (s *Server) handleRespondDispatcherRequest(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request id")
		return
	}

	var req respondDispatcherRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	updated, err := s.DispatcherRequests.Respond(r.Context(), id, callerID, req.Approve)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "request not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to respond to request: "+err.Error())
		return
	}

	resp := respondDispatcherRequestResponse{Status: updated.Status}
	if req.Approve {
		resp.DispatcherID = &updated.DispatcherID
	}
	writeJSON(w, http.StatusOK, resp)
}
