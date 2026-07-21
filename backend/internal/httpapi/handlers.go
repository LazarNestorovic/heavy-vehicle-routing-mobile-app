package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"heavy-vehicle-routing/backend/internal/valhalla"
)

type Server struct {
	Valhalla *valhalla.Client
}

func NewServer(v *valhalla.Client) *Server {
	return &Server{Valhalla: v}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /api/v1/routes", s.handleCreateRoute)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

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

type createRouteRequest struct {
	Origin      valhalla.LatLon `json:"origin"`
	Destination valhalla.LatLon `json:"destination"`
	Vehicle     vehicleProfile  `json:"vehicle"`
}

type createRouteResponse struct {
	DistanceKm  float64 `json:"distance_km"`
	DurationMin float64 `json:"duration_min"`
	Shape       string  `json:"shape"`
}

func (s *Server) handleCreateRoute(w http.ResponseWriter, r *http.Request) {
	var req createRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := (vehicleProfileValidator{}).Validate(req.Vehicle); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	profile := valhalla.TruckProfile{
		HeightM:    req.Vehicle.HeightM,
		WidthM:     req.Vehicle.WidthM,
		LengthM:    req.Vehicle.LengthM,
		WeightKg:   req.Vehicle.WeightKg,
		AxleLoadKg: req.Vehicle.AxleLoadKg,
		Hazmat:     req.Vehicle.Hazmat,
	}

	result, err := s.Valhalla.Route(r.Context(), req.Origin, req.Destination, profile)
	if err != nil {
		// No viable route for these vehicle constraints is a valid, meaningful outcome
		// (that's the whole point of vehicle-aware routing), not a server error.
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, createRouteResponse{
		DistanceKm:  result.DistanceKm,
		DurationMin: result.DurationMin,
		Shape:       result.Shape,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
