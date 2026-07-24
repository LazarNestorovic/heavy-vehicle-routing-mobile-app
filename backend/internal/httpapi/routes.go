package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"heavy-vehicle-routing/backend/internal/scoring"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

type createRouteRequest struct {
	Origin      valhalla.LatLon `json:"origin"`
	Destination valhalla.LatLon `json:"destination"`
	Vehicle     vehicleProfile  `json:"vehicle"`
}

type candidateResponse struct {
	DistanceKm    float64 `json:"distance_km"`
	DurationMin   float64 `json:"duration_min"`
	RiskScore     float64 `json:"risk_score"`
	ManeuverCount int     `json:"maneuver_count"`
	HighwayRatio  float64 `json:"highway_ratio"`
	HasFerry      bool    `json:"has_ferry"`
	HasToll       bool    `json:"has_toll"`
	Chosen        bool    `json:"chosen"`
}

type createRouteResponse struct {
	DistanceKm  float64             `json:"distance_km"`
	DurationMin float64             `json:"duration_min"`
	Shape       string              `json:"shape"`
	RiskScore   float64             `json:"risk_score"`
	Candidates  []candidateResponse `json:"candidates"`
	Explanation *string             `json:"explanation,omitempty"`
}

func toTruckProfile(profile vehicleProfile) valhalla.TruckProfile {
	return valhalla.TruckProfile{
		HeightM:    profile.HeightM,
		WidthM:     profile.WidthM,
		LengthM:    profile.LengthM,
		WeightKg:   profile.WeightKg,
		AxleLoadKg: profile.AxleLoadKg,
		Hazmat:     profile.Hazmat,
	}
}

// scoringPreferences loads the authenticated driver's saved priorities (or the
// neutral defaults if they've never set any - see store.PreferencesStore.Get).
func (s *Server) scoringPreferences(ctx context.Context, driverID int64) (scoring.Preferences, error) {
	prefs, err := s.Preferences.Get(ctx, driverID)
	if err != nil {
		return scoring.Preferences{}, err
	}
	return scoring.Preferences{
		FuelPriority: prefs.FuelPriority, CargoPriority: prefs.CargoPriority,
		HighwayPriority: prefs.HighwayPriority, TimePriority: prefs.TimePriority,
	}, nil
}

// resolvePreferredStops turns the driver's saved preferences (favorite stops +
// preferred fuel brand) into plain coordinates for scoring.Rank - scoring
// itself doesn't know about brands/favorites, only "is this route near one of
// these points".
func (s *Server) resolvePreferredStops(ctx context.Context, driverID int64) ([]valhalla.LatLon, error) {
	prefs, err := s.Preferences.Get(ctx, driverID)
	if err != nil {
		return nil, err
	}
	favorites, err := s.FavoriteStops.List(ctx, driverID)
	if err != nil {
		return nil, err
	}

	var points []valhalla.LatLon
	for _, f := range favorites {
		points = append(points, valhalla.LatLon{Lat: f.Lat, Lon: f.Lon})
	}
	if prefs.PreferredFuelBrand != nil {
		for _, stop := range s.RestStops.ByBrand(*prefs.PreferredFuelBrand) {
			points = append(points, valhalla.LatLon{Lat: stop.Lat, Lon: stop.Lon})
		}
	}
	return points, nil
}

// bestRoute requests alternatives from Valhalla for the given vehicle profile and
// returns them ranked by risk score against prefs (best first). Shared by
// /api/v1/routes (stateless preview) and /api/v1/trips (persisted).
func (s *Server) bestRoute(ctx context.Context, origin, destination valhalla.LatLon, profile vehicleProfile, prefs scoring.Preferences, preferredStops []valhalla.LatLon) ([]scoring.ScoredCandidate, error) {
	candidates, err := s.Valhalla.RouteAlternates(ctx, origin, destination, toTruckProfile(profile), numAlternates)
	if err != nil {
		return nil, err
	}
	return scoring.Rank(candidates, prefs, profile.WeightKg, preferredStops), nil
}

func toCandidateResponses(ranked []scoring.ScoredCandidate) []candidateResponse {
	out := make([]candidateResponse, len(ranked))
	for i, c := range ranked {
		out[i] = candidateResponse{
			DistanceKm:    c.DistanceKm,
			DurationMin:   c.DurationMin,
			RiskScore:     c.RiskScore,
			ManeuverCount: c.ManeuverCount,
			HighwayRatio:  c.HighwayRatio,
			HasFerry:      c.HasFerry,
			HasToll:       c.HasToll,
			Chosen:        i == 0,
		}
	}
	return out
}

func (s *Server) handleCreateRoute(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	var req createRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := (vehicleProfileValidator{}).Validate(req.Vehicle); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	prefs, err := s.scoringPreferences(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load preferences: "+err.Error())
		return
	}
	preferredStops, err := s.resolvePreferredStops(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load preferred stops: "+err.Error())
		return
	}

	ranked, err := s.bestRoute(r.Context(), req.Origin, req.Destination, req.Vehicle, prefs, preferredStops)
	if err != nil {
		// No viable route for these vehicle constraints is a valid, meaningful outcome
		// (that's the whole point of vehicle-aware routing), not a server error.
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	best := ranked[0]
	explanation := s.Explain.Explain(r.Context(), req.Origin, req.Destination, toTruckProfile(req.Vehicle), best.RouteCandidate, prefs, req.Vehicle.WeightKg, preferredStops)

	writeJSON(w, http.StatusOK, createRouteResponse{
		DistanceKm:  best.DistanceKm,
		DurationMin: best.DurationMin,
		Shape:       best.Shape,
		RiskScore:   best.RiskScore,
		Candidates:  toCandidateResponses(ranked),
		Explanation: explanation,
	})
}
