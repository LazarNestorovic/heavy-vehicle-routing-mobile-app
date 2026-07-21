package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

type createTripRequest struct {
	VehicleID   int64           `json:"vehicle_id"`
	Origin      valhalla.LatLon `json:"origin"`
	Destination valhalla.LatLon `json:"destination"`
}

type tripResponse struct {
	ID          int64               `json:"id"`
	VehicleID   int64               `json:"vehicle_id"`
	Status      string              `json:"status"`
	DistanceKm  float64             `json:"distance_km"`
	DurationMin float64             `json:"duration_min"`
	Shape       string              `json:"shape"`
	RiskScore   float64             `json:"risk_score"`
	Candidates  []candidateResponse `json:"candidates"`
}

// handleCreateTrip looks up a previously saved vehicle profile, computes and scores a
// route for it (same logic as POST /api/v1/routes), and persists the result as a trip.
func (s *Server) handleCreateTrip(w http.ResponseWriter, r *http.Request) {
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

	ranked, err := s.bestRoute(r.Context(), req.Origin, req.Destination, fromStoreVehicle(vehicle))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	best := ranked[0]

	saved, err := s.Trips.Create(r.Context(), store.Trip{
		VehicleID:      req.VehicleID,
		OriginLat:      req.Origin.Lat,
		OriginLon:      req.Origin.Lon,
		DestinationLat: req.Destination.Lat,
		DestinationLon: req.Destination.Lon,
		DistanceKm:     best.DistanceKm,
		DurationMin:    best.DurationMin,
		RiskScore:      best.RiskScore,
		Shape:          best.Shape,
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

	writeJSON(w, http.StatusCreated, tripResponse{
		ID:          saved.ID,
		VehicleID:   saved.VehicleID,
		Status:      saved.Status,
		DistanceKm:  saved.DistanceKm,
		DurationMin: saved.DurationMin,
		Shape:       saved.Shape,
		RiskScore:   saved.RiskScore,
		Candidates:  toCandidateResponses(ranked),
	})
}
