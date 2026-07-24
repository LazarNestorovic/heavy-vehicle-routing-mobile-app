package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"heavy-vehicle-routing/backend/internal/store"
)

type preferencesRequest struct {
	FuelPriority       int     `json:"fuel_priority"`
	CargoPriority      int     `json:"cargo_priority"`
	HighwayPriority    int     `json:"highway_priority"`
	TimePriority       int     `json:"time_priority"`
	PreferredFuelBrand *string `json:"preferred_fuel_brand,omitempty"`
}

func (req preferencesRequest) validate() error {
	for name, v := range map[string]int{
		"fuel_priority": req.FuelPriority, "cargo_priority": req.CargoPriority,
		"highway_priority": req.HighwayPriority, "time_priority": req.TimePriority,
	} {
		if v < 1 || v > 5 {
			return fmt.Errorf("%s must be between 1 and 5, got %d", name, v)
		}
	}
	return nil
}

func toStorePreferences(driverID int64, req preferencesRequest) store.DriverPreferences {
	return store.DriverPreferences{
		DriverID: driverID, FuelPriority: req.FuelPriority, CargoPriority: req.CargoPriority,
		HighwayPriority: req.HighwayPriority, TimePriority: req.TimePriority, PreferredFuelBrand: req.PreferredFuelBrand,
	}
}

func fromStorePreferences(p store.DriverPreferences) preferencesRequest {
	return preferencesRequest{
		FuelPriority: p.FuelPriority, CargoPriority: p.CargoPriority,
		HighwayPriority: p.HighwayPriority, TimePriority: p.TimePriority, PreferredFuelBrand: p.PreferredFuelBrand,
	}
}

func (s *Server) handleGetPreferences(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	prefs, err := s.Preferences.Get(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load preferences: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, fromStorePreferences(prefs))
}

func (s *Server) handleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	var req preferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	saved, err := s.Preferences.Upsert(r.Context(), toStorePreferences(driverID, req))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save preferences: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, fromStorePreferences(saved))
}
