package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

type createTripRequest struct {
	VehicleID   int64           `json:"vehicle_id"`
	Origin      valhalla.LatLon `json:"origin"`
	Destination valhalla.LatLon `json:"destination"`
}

type restStopResponse struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Name    string  `json:"name,omitempty"`
	Amenity string  `json:"amenity"`
}

type tripResponse struct {
	ID                    int64               `json:"id"`
	VehicleID             int64               `json:"vehicle_id"`
	Status                string              `json:"status"`
	DistanceKm            float64             `json:"distance_km"`
	DurationMin           float64             `json:"duration_min"`
	Shape                 string              `json:"shape"`
	RiskScore             float64             `json:"risk_score"`
	Candidates            []candidateResponse `json:"candidates,omitempty"`
	Explanation           *string             `json:"explanation,omitempty"`
	NextRestSuggestionMin *float64            `json:"next_rest_suggestion_min,omitempty"`
	RestStop              *restStopResponse   `json:"rest_stop,omitempty"`
}

func toTripResponse(t store.Trip, candidates []candidateResponse) tripResponse {
	resp := tripResponse{
		ID:                    t.ID,
		VehicleID:             t.VehicleID,
		Status:                t.Status,
		DistanceKm:            t.DistanceKm,
		DurationMin:           t.DurationMin,
		Shape:                 t.Shape,
		RiskScore:             t.RiskScore,
		Candidates:            candidates,
		Explanation:           t.Explanation,
		NextRestSuggestionMin: t.NextRestSuggestionMin,
	}
	if t.RestStopLat != nil && t.RestStopLon != nil {
		stop := restStopResponse{Lat: *t.RestStopLat, Lon: *t.RestStopLon}
		if t.RestStopName != nil {
			stop.Name = *t.RestStopName
		}
		if t.RestStopAmenity != nil {
			stop.Amenity = *t.RestStopAmenity
		}
		resp.RestStop = &stop
	}
	return resp
}

// handleCreateTrip looks up a previously saved vehicle profile, computes and scores a
// route for it (same logic as POST /api/v1/routes), and persists the result as a trip.
func (s *Server) handleCreateTrip(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	var req createTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	vehicle, err := s.Vehicles.Get(r.Context(), req.VehicleID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "vehicle not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load vehicle: "+err.Error())
		return
	}
	if vehicle.DriverID != driverID {
		writeError(w, http.StatusForbidden, "vehicle does not belong to you")
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

	profile := fromStoreVehicle(vehicle)
	ranked, err := s.bestRoute(r.Context(), req.Origin, req.Destination, profile, prefs, preferredStops)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	best := ranked[0]
	explanation := s.Explain.Explain(r.Context(), req.Origin, req.Destination, toTruckProfile(profile), best.RouteCandidate, prefs, profile.WeightKg, preferredStops)

	saved, err := s.Trips.Create(r.Context(), store.Trip{
		DriverID:       driverID,
		VehicleID:      req.VehicleID,
		OriginLat:      req.Origin.Lat,
		OriginLon:      req.Origin.Lon,
		DestinationLat: req.Destination.Lat,
		DestinationLon: req.Destination.Lon,
		DistanceKm:     best.DistanceKm,
		DurationMin:    best.DurationMin,
		RiskScore:      best.RiskScore,
		Shape:          best.Shape,
		Explanation:    explanation,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save trip: "+err.Error())
		return
	}

	// The trip is already persisted (source of truth); a queue hiccup here shouldn't
	// fail the request, so we log and move on rather than returning an error.
	if body, err := json.Marshal(queue.TripStartedEvent{TripID: saved.ID}); err != nil {
		log.Printf("encode trip.started for trip %d: %v", saved.ID, err)
	} else if err := s.Queue.Publish(r.Context(), queue.RoutingKeyTripStarted, body); err != nil {
		log.Printf("publish trip.started for trip %d: %v", saved.ID, err)
	}

	writeJSON(w, http.StatusCreated, toTripResponse(saved, toCandidateResponses(ranked)))
}

// handleGetTrip returns the current state of a trip, including the rest-stop
// suggestion once the trip.started worker has processed it (status "in_progress").
func (s *Server) handleGetTrip(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid trip id")
		return
	}

	trip, err := s.Trips.Get(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "trip not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load trip: "+err.Error())
		return
	}
	if trip.DriverID != driverID {
		writeError(w, http.StatusForbidden, "trip does not belong to you")
		return
	}

	writeJSON(w, http.StatusOK, toTripResponse(trip, nil))
}

// handleTripStream checks trip ownership (the ws package itself knows nothing
// about auth) before delegating to the WebSocket gateway.
func (s *Server) handleTripStream(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid trip id")
		return
	}

	trip, err := s.Trips.Get(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "trip not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load trip: "+err.Error())
		return
	}
	if trip.DriverID != driverID {
		writeError(w, http.StatusForbidden, "trip does not belong to you")
		return
	}

	s.WS.HandleTripStream(w, r)
}
