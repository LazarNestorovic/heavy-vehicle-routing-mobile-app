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
	ID int64 `json:"id"`
	vehicleProfile
}

func (s *Server) handleCreateVehicle(w http.ResponseWriter, r *http.Request) {
	var req vehicleProfile
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := (vehicleProfileValidator{}).Validate(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	saved, err := s.Vehicles.Create(r.Context(), toStoreVehicle(req))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save vehicle: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, vehicleResponse{ID: saved.ID, vehicleProfile: fromStoreVehicle(saved)})
}

func (s *Server) handleGetVehicle(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, vehicleResponse{ID: v.ID, vehicleProfile: fromStoreVehicle(v)})
}
