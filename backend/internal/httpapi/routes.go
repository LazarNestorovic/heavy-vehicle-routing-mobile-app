package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
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
	// Shape is the encoded polyline6 for THIS candidate (not just the chosen
	// one) - lets the client draw not-chosen alternatives on the map (see
	// documentations/features/ entry), same encoding as the top-level shape.
	Shape string `json:"shape"`
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

// preferredStop pairs a coordinate with a display name - scoring.Rank only
// needs the coordinate (see plainCoords), but preferredStopMessage needs the
// name too, to say WHICH favorite/brand stop a route passed near.
type preferredStop struct {
	valhalla.LatLon
	Name string
}

// resolvePreferredStops turns the driver's saved preferences (favorite stops +
// preferred fuel brand) into named points.
func (s *Server) resolvePreferredStops(ctx context.Context, driverID int64) ([]preferredStop, error) {
	prefs, err := s.Preferences.Get(ctx, driverID)
	if err != nil {
		return nil, err
	}
	favorites, err := s.FavoriteStops.List(ctx, driverID)
	if err != nil {
		return nil, err
	}

	var stops []preferredStop
	for _, f := range favorites {
		stops = append(stops, preferredStop{LatLon: valhalla.LatLon{Lat: f.Lat, Lon: f.Lon}, Name: f.Name})
	}
	if prefs.PreferredFuelBrand != nil {
		for _, stop := range s.RestStops.ByBrand(*prefs.PreferredFuelBrand) {
			name := stop.Name
			if name == "" {
				name = stop.Brand
			}
			stops = append(stops, preferredStop{LatLon: valhalla.LatLon{Lat: stop.Lat, Lon: stop.Lon}, Name: name})
		}
	}
	return stops, nil
}

// plainCoords strips names for the callers (scoring.Rank, internal/explain)
// that only care about coordinates.
func plainCoords(stops []preferredStop) []valhalla.LatLon {
	out := make([]valhalla.LatLon, len(stops))
	for i, s := range stops {
		out[i] = s.LatLon
	}
	return out
}

// preferredStopMessage builds a driver-facing "why" note when shape passes
// near one of stops - the counterpart to Explain's vehicle-constraint
// explanation, for the OTHER reason a route's explanation field might be
// interesting: not a restriction that forced a detour, but a favorite/brand
// stop that earned the route a scoring bonus (see scoring.go
// preferredStopDiscount). Only called when Explain itself found nothing (see
// call sites) - the two never stack into one message.
func preferredStopMessage(shape string, stops []preferredStop) *string {
	matched, ok := scoring.NearestPreferredStopWithinRadius(shape, plainCoords(stops))
	if !ok {
		return nil
	}
	name := ""
	for _, s := range stops {
		if s.LatLon == matched {
			name = s.Name
			break
		}
	}
	var msg string
	if name != "" {
		msg = fmt.Sprintf("Ruta prolazi blizu vaše omiljene pumpe %q.", name)
	} else {
		msg = "Ruta prolazi blizu jedne od vaših omiljenih pumpi."
	}
	return &msg
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
			Shape:         c.Shape,
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

	ranked, err := s.bestRoute(r.Context(), req.Origin, req.Destination, req.Vehicle, prefs, plainCoords(preferredStops))
	if err != nil {
		// No viable route for these vehicle constraints is a valid, meaningful outcome
		// (that's the whole point of vehicle-aware routing), not a server error.
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	best := ranked[0]
	explanation := s.Explain.Explain(r.Context(), req.Origin, req.Destination, toTruckProfile(req.Vehicle), best.RouteCandidate, prefs, req.Vehicle.WeightKg, plainCoords(preferredStops))
	if explanation == nil {
		explanation = preferredStopMessage(best.Shape, preferredStops)
	}

	writeJSON(w, http.StatusOK, createRouteResponse{
		DistanceKm:  best.DistanceKm,
		DurationMin: best.DurationMin,
		Shape:       best.Shape,
		RiskScore:   best.RiskScore,
		Candidates:  toCandidateResponses(ranked),
		Explanation: explanation,
	})
}
