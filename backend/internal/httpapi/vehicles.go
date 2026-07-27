package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"heavy-vehicle-routing/backend/internal/store"
)

type vehicleProfile struct {
	HeightM    float64 `json:"height_m"`
	WidthM     float64 `json:"width_m"`
	LengthM    float64 `json:"length_m"`
	WeightKg   float64 `json:"weight_kg"`
	AxleLoadKg float64 `json:"axle_load_kg"`
	Hazmat     bool    `json:"hazmat"`
}

// vehicleProfileValidator checks a vehicleProfile for physically inconsistent values
// that Valhalla itself doesn't reject (it treats each costing field independently).
type vehicleProfileValidator struct{}

func (vehicleProfileValidator) Validate(v vehicleProfile) error {
	if v.AxleLoadKg > v.WeightKg {
		return fmt.Errorf("axle_load_kg (%.0f) cannot be greater than weight_kg (%.0f)", v.AxleLoadKg, v.WeightKg)
	}
	return nil
}

func toStoreVehicle(v vehicleProfile) store.Vehicle {
	return store.Vehicle{
		HeightM:    v.HeightM,
		WidthM:     v.WidthM,
		LengthM:    v.LengthM,
		WeightKg:   v.WeightKg,
		AxleLoadKg: v.AxleLoadKg,
		Hazmat:     v.Hazmat,
	}
}

func fromStoreVehicle(v store.Vehicle) vehicleProfile {
	return vehicleProfile{
		HeightM:    v.HeightM,
		WidthM:     v.WidthM,
		LengthM:    v.LengthM,
		WeightKg:   v.WeightKg,
		AxleLoadKg: v.AxleLoadKg,
		Hazmat:     v.Hazmat,
	}
}

type vehicleResponse struct {
	ID            int64    `json:"id"`
	FuelPercent   float64  `json:"fuel_percent"`
	NextServiceKm *float64 `json:"next_service_km,omitempty"`
	// IsFleet distinguishes a dispatcher's fleet vehicle from a driver's
	// personal one - lets a client group/label a mixed list (e.g. the
	// dispatcher create-trip picker, see documentations/features/ entry).
	IsFleet bool `json:"is_fleet"`
	vehicleProfile
}

func toVehicleResponse(v store.Vehicle) vehicleResponse {
	return vehicleResponse{
		ID: v.ID, FuelPercent: v.FuelPercent, NextServiceKm: v.NextServiceKm,
		IsFleet:        v.DispatcherID != nil,
		vehicleProfile: fromStoreVehicle(v),
	}
}

func (s *Server) handleCreateVehicle(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	account, err := s.loadAccount(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}

	var req vehicleProfile
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := (vehicleProfileValidator{}).Validate(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	toSave := toStoreVehicle(req)
	// A dispatcher's vehicles are fleet vehicles; everyone else's are personal
	// (independent drivers and managed drivers alike).
	if account.Role == store.RoleDispatcher {
		toSave.DispatcherID = &driverID
	} else {
		toSave.DriverID = &driverID
	}

	saved, err := s.Vehicles.Create(r.Context(), toSave)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save vehicle: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toVehicleResponse(saved))
}

func (s *Server) handleListVehicles(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	account, err := s.loadAccount(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}

	var vehicles []store.Vehicle
	if account.Role == store.RoleDispatcher {
		vehicles, err = s.Vehicles.ListFleet(r.Context(), driverID)
	} else {
		vehicles, err = s.Vehicles.List(r.Context(), driverID)
		if err == nil && account.DispatcherID != nil {
			fleet, ferr := s.Vehicles.ListFleet(r.Context(), *account.DispatcherID)
			if ferr != nil {
				err = ferr
			} else {
				vehicles = append(vehicles, fleet...)
			}
		}
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list vehicles: "+err.Error())
		return
	}

	out := make([]vehicleResponse, len(vehicles))
	for i, v := range vehicles {
		out[i] = toVehicleResponse(v)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleListDriverVehiclesForDispatcher lets a dispatcher see a managed
// driver's OWN personal vehicles too, not just their own fleet - the
// dispatcher's create-trip picker only offered fleet vehicles before (see
// documentations/features/ entry), even though POST /trips already accepted
// a driver's personal vehicle for an assigned trip.
func (s *Server) handleListDriverVehiclesForDispatcher(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())
	if _, ok := s.requireDispatcher(w, r, callerID); !ok {
		return
	}

	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid driver id")
		return
	}

	target, err := s.Drivers.Get(r.Context(), targetID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load driver: "+err.Error())
		return
	}
	if target.DispatcherID == nil || *target.DispatcherID != callerID {
		writeError(w, http.StatusForbidden, "driver is not managed by you")
		return
	}

	fleet, err := s.Vehicles.ListFleet(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list fleet vehicles: "+err.Error())
		return
	}
	personal, err := s.Vehicles.List(r.Context(), targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list driver vehicles: "+err.Error())
		return
	}

	out := make([]vehicleResponse, 0, len(fleet)+len(personal))
	for _, v := range fleet {
		out = append(out, toVehicleResponse(v))
	}
	for _, v := range personal {
		out = append(out, toVehicleResponse(v))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetVehicle(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	v, err := s.Vehicles.Get(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "vehicle not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load vehicle: "+err.Error())
		return
	}

	account, err := s.loadAccount(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}
	if !vehicleAccessible(v, account) {
		writeError(w, http.StatusForbidden, "vehicle does not belong to you")
		return
	}

	writeJSON(w, http.StatusOK, toVehicleResponse(v))
}

// handleUpdateVehicle edits a vehicle's physical dimensions - same ownership
// check as handleUpdateVehicleStatus, but for the profile fields instead of
// the fuel/service status fields.
func (s *Server) handleUpdateVehicle(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	v, err := s.Vehicles.Get(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "vehicle not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load vehicle: "+err.Error())
		return
	}

	account, err := s.loadAccount(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}
	if !vehicleMutable(v, account) {
		writeError(w, http.StatusForbidden, "vehicle does not belong to you")
		return
	}

	var req vehicleProfile
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if err := (vehicleProfileValidator{}).Validate(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated := toStoreVehicle(req)
	if err := s.Vehicles.Update(r.Context(), id, updated); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update vehicle: "+err.Error())
		return
	}

	v.HeightM, v.WidthM, v.LengthM = req.HeightM, req.WidthM, req.LengthM
	v.WeightKg, v.AxleLoadKg, v.Hazmat = req.WeightKg, req.AxleLoadKg, req.Hazmat
	writeJSON(w, http.StatusOK, toVehicleResponse(v))
}

// handleDeleteVehicle removes a vehicle - 409 if any trip still references it
// (store.ErrVehicleInUse; trips are an append-only historical record, so
// deleting a vehicle never cascades into deleting trips).
func (s *Server) handleDeleteVehicle(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	v, err := s.Vehicles.Get(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "vehicle not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load vehicle: "+err.Error())
		return
	}

	account, err := s.loadAccount(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}
	if !vehicleMutable(v, account) {
		writeError(w, http.StatusForbidden, "vehicle does not belong to you")
		return
	}

	if err := s.Vehicles.Delete(r.Context(), id); err != nil {
		if err == store.ErrVehicleInUse {
			writeError(w, http.StatusConflict, "vehicle has trips associated with it and cannot be deleted")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete vehicle: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateVehicleStatusRequest struct {
	FuelPercent   float64  `json:"fuel_percent"`
	NextServiceKm *float64 `json:"next_service_km,omitempty"`
}

func (s *Server) handleUpdateVehicleStatus(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	v, err := s.Vehicles.Get(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "vehicle not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load vehicle: "+err.Error())
		return
	}

	account, err := s.loadAccount(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}
	if !vehicleAccessible(v, account) {
		writeError(w, http.StatusForbidden, "vehicle does not belong to you")
		return
	}

	var req updateVehicleStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := s.Vehicles.UpdateStatus(r.Context(), id, req.FuelPercent, req.NextServiceKm); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update vehicle status: "+err.Error())
		return
	}

	v.FuelPercent = req.FuelPercent
	v.NextServiceKm = req.NextServiceKm
	writeJSON(w, http.StatusOK, toVehicleResponse(v))
}

type vehicleHoursResponse struct {
	SinceLastBreakMin float64 `json:"since_last_break_min"`
	DrivingTodayMin   float64 `json:"driving_today_min"`
}

func (s *Server) handleGetVehicleHours(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	v, err := s.Vehicles.Get(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "vehicle not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load vehicle: "+err.Error())
		return
	}

	account, err := s.loadAccount(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}
	if !vehicleAccessible(v, account) {
		writeError(w, http.StatusForbidden, "vehicle does not belong to you")
		return
	}

	hours, err := s.Trips.DrivingHours(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute driving hours: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, vehicleHoursResponse{
		SinceLastBreakMin: hours.SinceLastBreakMin,
		DrivingTodayMin:   hours.DrivingTodayMin,
	})
}
