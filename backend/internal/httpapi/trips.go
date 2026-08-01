package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

type createTripRequest struct {
	VehicleID        int64           `json:"vehicle_id"`
	Origin           valhalla.LatLon `json:"origin"`
	Destination      valhalla.LatLon `json:"destination"`
	CargoDescription *string         `json:"cargo_description,omitempty"`
	CargoWeightKg    *float64        `json:"cargo_weight_kg,omitempty"`
	CargoTempRange   *string         `json:"cargo_temp_range,omitempty"`
	PickupLocation   *string         `json:"pickup_location,omitempty"`
	DropoffLocation  *string         `json:"dropoff_location,omitempty"`
	// DriverID is required when the caller is a dispatcher (who this trip is
	// assigned to); ignored/must be empty for self-service creation.
	DriverID *int64 `json:"driver_id,omitempty"`
}

type restStopResponse struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Name    string  `json:"name,omitempty"`
	Amenity string  `json:"amenity"`
}

type tripResponse struct {
	ID                    int64               `json:"id"`
	DriverID              int64               `json:"driver_id"`
	DriverUsername        string              `json:"driver_username,omitempty"`
	VehicleID             int64               `json:"vehicle_id"`
	Status                string              `json:"status"`
	Origin                valhalla.LatLon     `json:"origin"`
	Destination           valhalla.LatLon     `json:"destination"`
	DistanceKm            float64             `json:"distance_km"`
	DurationMin           float64             `json:"duration_min"`
	Shape                 string              `json:"shape"`
	RiskScore             float64             `json:"risk_score"`
	Candidates            []candidateResponse `json:"candidates,omitempty"`
	Explanation           *string             `json:"explanation,omitempty"`
	NextRestSuggestionMin *float64            `json:"next_rest_suggestion_min,omitempty"`
	RestStop              *restStopResponse   `json:"rest_stop,omitempty"`
	CargoDescription      *string             `json:"cargo_description,omitempty"`
	CargoWeightKg         *float64            `json:"cargo_weight_kg,omitempty"`
	CargoTempRange        *string             `json:"cargo_temp_range,omitempty"`
	PickupLocation        *string             `json:"pickup_location,omitempty"`
	DropoffLocation       *string             `json:"dropoff_location,omitempty"`
	CreatedAt             time.Time           `json:"created_at"`
}

func toTripResponse(t store.Trip, candidates []candidateResponse) tripResponse {
	resp := tripResponse{
		ID:                    t.ID,
		DriverID:              t.DriverID,
		VehicleID:             t.VehicleID,
		Status:                t.Status,
		Origin:                valhalla.LatLon{Lat: t.OriginLat, Lon: t.OriginLon},
		Destination:           valhalla.LatLon{Lat: t.DestinationLat, Lon: t.DestinationLon},
		DistanceKm:            t.DistanceKm,
		DurationMin:           t.DurationMin,
		Shape:                 t.Shape,
		RiskScore:             t.RiskScore,
		Candidates:            candidates,
		Explanation:           t.Explanation,
		NextRestSuggestionMin: t.NextRestSuggestionMin,
		CargoDescription:      t.CargoDescription,
		CargoWeightKg:         t.CargoWeightKg,
		CargoTempRange:        t.CargoTempRange,
		PickupLocation:        t.PickupLocation,
		DropoffLocation:       t.DropoffLocation,
		CreatedAt:             t.CreatedAt,
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

// startTripSideEffects logs the departed event and publishes trip.started -
// shared by self-service creation (fires immediately) and the dispatcher-
// offer flow (fires only once the driver clicks "start", see handleStartTrip).
func (s *Server) startTripSideEffects(ctx context.Context, tripID int64) {
	if _, err := s.TripEvents.Create(ctx, tripID, "departed", "Departed"); err != nil {
		log.Printf("log departed event for trip %d: %v", tripID, err)
	}
	// The trip is already persisted (source of truth); a queue hiccup here shouldn't
	// fail the request, so we log and move on rather than returning an error.
	if body, err := json.Marshal(queue.TripStartedEvent{TripID: tripID}); err != nil {
		log.Printf("encode trip.started for trip %d: %v", tripID, err)
	} else if err := s.Queue.Publish(ctx, queue.RoutingKeyTripStarted, body); err != nil {
		log.Printf("publish trip.started for trip %d: %v", tripID, err)
	}
}

// handleCreateTrip computes and scores a route (same logic as POST /api/v1/routes)
// and persists it as a trip. Behavior branches by caller role:
//   - Independent driver (no dispatcher): unchanged self-service flow, trip starts
//     immediately (status "created", departed event fires now) - rejected with
//     409 if they already have another trip active (see HasActiveTrip).
//   - Managed driver (has a dispatcher): self-service is allowed, but only for
//     their OWN personal vehicle - their dispatcher's fleet trucks still go
//     through the dispatcher (rejected otherwise). Also rejected with 409 if
//     they have a pending/accepted request from their dispatcher (see
//     HasPendingOffer) or an own trip already under way (see HasActiveTrip,
//     same check as the independent-driver path).
//   - Dispatcher: req.DriverID selects which of their managed drivers this trip is
//     for; the vehicle may be the dispatcher's own fleet OR that driver's personal
//     vehicle. Trip is saved as "offered" - departed/trip.started fire later, from
//     handleStartTrip, once the driver actually starts it. Preferences/favorite
//     stops used for scoring are always the CALLER's own (see documentations/
//     features/ entry - the dispatcher's own profile stands in for "company
//     defaults", not the assigned driver's).
func (s *Server) handleCreateTrip(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())

	caller, err := s.loadAccount(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}

	var req createTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	assignedDriverID := callerID
	var assignedByID *int64
	status := ""
	if caller.Role == store.RoleDispatcher {
		if req.DriverID == nil {
			writeError(w, http.StatusBadRequest, "driver_id is required")
			return
		}
		target, err := s.Drivers.Get(r.Context(), *req.DriverID)
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
		assignedDriverID = *req.DriverID
		assignedByID = &callerID
		status = store.TripStatusOffered
	} else if caller.DispatcherID != nil {
		// Managed driver: self-service is allowed for their OWN personal
		// vehicle - their dispatcher's fleet trucks still go through the
		// dispatcher. Even then, blocked while a pending/accepted request
		// from the dispatcher exists; an own trip already under way is
		// caught below by HasActiveTrip, same as the independent-driver path.
		v, err := s.Vehicles.Get(r.Context(), req.VehicleID)
		if err != nil {
			if err == store.ErrNotFound {
				writeError(w, http.StatusNotFound, "vehicle not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to load vehicle: "+err.Error())
			return
		}
		if v.DriverID == nil || *v.DriverID != callerID {
			writeError(w, http.StatusForbidden, "your dispatcher creates trips for fleet vehicles")
			return
		}
		pending, err := s.Trips.HasPendingOffer(r.Context(), callerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check pending offer: "+err.Error())
			return
		}
		if pending != nil {
			writeError(w, http.StatusConflict, "you have a pending route request from your dispatcher")
			return
		}
	}

	// Self-service creation starts the trip immediately (status stays "") - a
	// driver already underway on one trip can't start a second at the same
	// time. A dispatcher's offer (status "offered") isn't gated here; the
	// driver hasn't committed to it yet, only actually STARTING one is
	// blocked (see handleStartTrip for the managed-driver equivalent).
	if status == "" {
		active, err := s.Trips.HasActiveTrip(r.Context(), assignedDriverID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check active trip: "+err.Error())
			return
		}
		if active != nil {
			writeError(w, http.StatusConflict, "you already have an active trip in progress")
			return
		}
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
	// Accessible to the caller directly (own personal/fleet vehicle), or - for a
	// dispatcher - the assigned driver's own personal vehicle.
	if !vehicleAccessible(vehicle, caller) && !(vehicle.DriverID != nil && *vehicle.DriverID == assignedDriverID) {
		writeError(w, http.StatusForbidden, "vehicle does not belong to you")
		return
	}

	prefs, err := s.scoringPreferences(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load preferences: "+err.Error())
		return
	}
	preferredStops, err := s.resolvePreferredStops(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load preferred stops: "+err.Error())
		return
	}

	profile := fromStoreVehicle(vehicle)
	ranked, err := s.bestRoute(r.Context(), req.Origin, req.Destination, profile, prefs, plainCoords(preferredStops))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	best := ranked[0]
	explanation := s.Explain.Explain(r.Context(), req.Origin, req.Destination, toTruckProfile(profile), best.RouteCandidate, prefs, profile.WeightKg, plainCoords(preferredStops))
	if explanation == nil {
		explanation = preferredStopMessage(best.Shape, preferredStops)
	}

	saved, err := s.Trips.Create(r.Context(), store.Trip{
		DriverID:         assignedDriverID,
		AssignedByID:     assignedByID,
		VehicleID:        req.VehicleID,
		OriginLat:        req.Origin.Lat,
		OriginLon:        req.Origin.Lon,
		DestinationLat:   req.Destination.Lat,
		DestinationLon:   req.Destination.Lon,
		DistanceKm:       best.DistanceKm,
		DurationMin:      best.DurationMin,
		RiskScore:        best.RiskScore,
		Shape:            best.Shape,
		Status:           status,
		Explanation:      explanation,
		CargoDescription: req.CargoDescription,
		CargoWeightKg:    req.CargoWeightKg,
		CargoTempRange:   req.CargoTempRange,
		PickupLocation:   req.PickupLocation,
		DropoffLocation:  req.DropoffLocation,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save trip: "+err.Error())
		return
	}

	if saved.Status != store.TripStatusOffered {
		s.startTripSideEffects(r.Context(), saved.ID)
	}

	writeJSON(w, http.StatusCreated, toTripResponse(saved, toCandidateResponses(ranked)))
}

type updateTripRequest struct {
	VehicleID        int64           `json:"vehicle_id"`
	Origin           valhalla.LatLon `json:"origin"`
	Destination      valhalla.LatLon `json:"destination"`
	CargoDescription *string         `json:"cargo_description,omitempty"`
	CargoWeightKg    *float64        `json:"cargo_weight_kg,omitempty"`
	CargoTempRange   *string         `json:"cargo_temp_range,omitempty"`
	PickupLocation   *string         `json:"pickup_location,omitempty"`
	DropoffLocation  *string         `json:"dropoff_location,omitempty"`
}

// handleUpdateTrip lets the dispatcher who assigned a trip change its
// vehicle/route/cargo while the driver hasn't departed yet - "offered" (not
// yet reviewed) or "accepted" (committed but not started). Editing an
// "accepted" trip reverts it to "offered" (see store.TripStore.
// EditByDispatcher): the trip changed after the driver committed to it, so
// they need to see the new version and decide again; an "offered" trip
// doesn't need a status change either way. Recomputes the route exactly like
// handleCreateTrip, since origin/destination/vehicle may all have changed.
// The assigned driver itself is NOT editable here - only the trip's content.
func (s *Server) handleUpdateTrip(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())

	caller, err := s.loadAccount(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}
	if caller.Role != store.RoleDispatcher {
		writeError(w, http.StatusForbidden, "only a dispatcher can edit a trip")
		return
	}

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
	if trip.AssignedByID == nil || *trip.AssignedByID != callerID {
		writeError(w, http.StatusForbidden, "trip does not belong to you")
		return
	}
	if trip.Status != store.TripStatusOffered && trip.Status != store.TripStatusAccepted {
		writeError(w, http.StatusConflict, "trip can only be edited while offered or accepted")
		return
	}

	var req updateTripRequest
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
	// Same accessibility rule as handleCreateTrip: the dispatcher's own fleet,
	// OR the assigned driver's own personal vehicle.
	if !vehicleAccessible(vehicle, caller) && !(vehicle.DriverID != nil && *vehicle.DriverID == trip.DriverID) {
		writeError(w, http.StatusForbidden, "vehicle does not belong to you")
		return
	}

	prefs, err := s.scoringPreferences(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load preferences: "+err.Error())
		return
	}
	preferredStops, err := s.resolvePreferredStops(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load preferred stops: "+err.Error())
		return
	}

	profile := fromStoreVehicle(vehicle)
	ranked, err := s.bestRoute(r.Context(), req.Origin, req.Destination, profile, prefs, plainCoords(preferredStops))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	best := ranked[0]
	explanation := s.Explain.Explain(r.Context(), req.Origin, req.Destination, toTruckProfile(profile), best.RouteCandidate, prefs, profile.WeightKg, plainCoords(preferredStops))
	if explanation == nil {
		explanation = preferredStopMessage(best.Shape, preferredStops)
	}

	wasAccepted := trip.Status == store.TripStatusAccepted

	if err := s.Trips.EditByDispatcher(r.Context(), trip.ID, store.Trip{
		VehicleID:        req.VehicleID,
		OriginLat:        req.Origin.Lat,
		OriginLon:        req.Origin.Lon,
		DestinationLat:   req.Destination.Lat,
		DestinationLon:   req.Destination.Lon,
		DistanceKm:       best.DistanceKm,
		DurationMin:      best.DurationMin,
		RiskScore:        best.RiskScore,
		Shape:            best.Shape,
		Explanation:      explanation,
		CargoDescription: req.CargoDescription,
		CargoWeightKg:    req.CargoWeightKg,
		CargoTempRange:   req.CargoTempRange,
		PickupLocation:   req.PickupLocation,
		DropoffLocation:  req.DropoffLocation,
	}); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusConflict, "trip is no longer editable")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update trip: "+err.Error())
		return
	}

	eventDesc := "Dispatcher updated the trip"
	if wasAccepted {
		eventDesc = "Dispatcher updated the trip - review needed again"
	}
	if _, err := s.TripEvents.Create(r.Context(), trip.ID, "edited", eventDesc); err != nil {
		log.Printf("log edited event for trip %d: %v", trip.ID, err)
	}

	updated, err := s.Trips.Get(r.Context(), trip.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload trip: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTripResponse(updated, toCandidateResponses(ranked)))
}

// loadOwnTrip loads trip id and checks it's assigned to driverID (strict -
// unlike tripAccessible, the dispatcher who assigned it may NOT accept/
// reject/start on the driver's behalf). Writes an error response and returns
// ok=false on any failure.
func (s *Server) loadOwnTrip(w http.ResponseWriter, r *http.Request, driverID int64) (store.Trip, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid trip id")
		return store.Trip{}, false
	}

	trip, err := s.Trips.Get(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "trip not found")
			return store.Trip{}, false
		}
		writeError(w, http.StatusInternalServerError, "failed to load trip: "+err.Error())
		return store.Trip{}, false
	}
	if trip.DriverID != driverID {
		writeError(w, http.StatusForbidden, "trip does not belong to you")
		return store.Trip{}, false
	}
	return trip, true
}

// handleAcceptTrip transitions an "offered" trip to "accepted" - the driver
// has reviewed the route/cargo/vehicle and committed to it, but hasn't
// departed yet (see handleStartTrip for that).
func (s *Server) handleAcceptTrip(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	trip, ok := s.loadOwnTrip(w, r, driverID)
	if !ok {
		return
	}
	if trip.Status != store.TripStatusOffered {
		writeError(w, http.StatusConflict, "trip is not in an offered state")
		return
	}

	if err := s.Trips.MarkAccepted(r.Context(), trip.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to accept trip: "+err.Error())
		return
	}
	trip.Status = store.TripStatusAccepted
	writeJSON(w, http.StatusOK, toTripResponse(trip, nil))
}

// handleRejectTrip transitions an "offered" trip to "rejected" - the driver
// declined it. Terminal state; the dispatcher can see it via GET /trips.
func (s *Server) handleRejectTrip(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	trip, ok := s.loadOwnTrip(w, r, driverID)
	if !ok {
		return
	}
	if trip.Status != store.TripStatusOffered {
		writeError(w, http.StatusConflict, "trip is not in an offered state")
		return
	}

	if err := s.Trips.MarkRejected(r.Context(), trip.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reject trip: "+err.Error())
		return
	}
	trip.Status = store.TripStatusRejected
	writeJSON(w, http.StatusOK, toTripResponse(trip, nil))
}

// handleStartTrip transitions an "accepted" trip to active, firing the same
// departed-event/trip.started side effects that self-service creation fires
// immediately.
func (s *Server) handleStartTrip(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	trip, ok := s.loadOwnTrip(w, r, driverID)
	if !ok {
		return
	}
	if trip.Status != store.TripStatusAccepted {
		writeError(w, http.StatusConflict, "trip is not in an accepted state")
		return
	}

	active, err := s.Trips.HasActiveTrip(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check active trip: "+err.Error())
		return
	}
	if active != nil {
		writeError(w, http.StatusConflict, "you already have an active trip in progress")
		return
	}

	if err := s.Trips.MarkStarted(r.Context(), trip.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start trip: "+err.Error())
		return
	}
	s.startTripSideEffects(r.Context(), trip.ID)

	trip.Status = store.TripStatusCreated
	writeJSON(w, http.StatusOK, toTripResponse(trip, nil))
}

type positionRequest struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type rerouteRequest struct {
	Origin valhalla.LatLon `json:"origin"`
}

// handleRerouteTrip recalculates and persists a trip's route from a NEW
// origin (typically the driver's current position, after they've deviated
// from the planned route - see mobile ActiveTripScreen's off-route
// detection) to its ORIGINAL, unchanged destination. Reuses the exact same
// scoring pipeline as handleCreateTrip for consistency, and re-publishes
// trip.started so the existing worker recomputes a rest-stop suggestion for
// the new route (Reroute() clears the old one, computed for a path that may
// no longer apply).
func (s *Server) handleRerouteTrip(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	trip, ok := s.loadOwnTrip(w, r, driverID)
	if !ok {
		return
	}
	if trip.Status != store.TripStatusCreated && trip.Status != store.TripStatusInProgress {
		writeError(w, http.StatusConflict, "trip is not currently active")
		return
	}

	var req rerouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	vehicle, err := s.Vehicles.Get(r.Context(), trip.VehicleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load vehicle: "+err.Error())
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

	destination := valhalla.LatLon{Lat: trip.DestinationLat, Lon: trip.DestinationLon}
	profile := fromStoreVehicle(vehicle)
	ranked, err := s.bestRoute(r.Context(), req.Origin, destination, profile, prefs, plainCoords(preferredStops))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	best := ranked[0]
	explanation := s.Explain.Explain(r.Context(), req.Origin, destination, toTruckProfile(profile), best.RouteCandidate, prefs, profile.WeightKg, plainCoords(preferredStops))
	if explanation == nil {
		explanation = preferredStopMessage(best.Shape, preferredStops)
	}

	if err := s.Trips.Reroute(r.Context(), trip.ID, req.Origin.Lat, req.Origin.Lon, best.DistanceKm, best.DurationMin, best.RiskScore, best.Shape, explanation); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reroute trip: "+err.Error())
		return
	}
	if _, err := s.TripEvents.Create(r.Context(), trip.ID, "rerouted", "Route recalculated after deviation"); err != nil {
		log.Printf("log rerouted event for trip %d: %v", trip.ID, err)
	}
	if body, err := json.Marshal(queue.TripStartedEvent{TripID: trip.ID}); err != nil {
		log.Printf("encode trip.started for reroute of trip %d: %v", trip.ID, err)
	} else if err := s.Queue.Publish(r.Context(), queue.RoutingKeyTripStarted, body); err != nil {
		log.Printf("publish trip.started for reroute of trip %d: %v", trip.ID, err)
	}

	updated, err := s.Trips.Get(r.Context(), trip.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload trip: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTripResponse(updated, toCandidateResponses(ranked)))
}

// handleReportPosition is called by the driver's own phone with a real GPS
// fix (see documentations/features/ live-GPS entry) - broadcasts it to this
// trip's WS watchers (the driver's own screen, and any dispatcher watching
// the fleet map).
func (s *Server) handleReportPosition(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	trip, ok := s.loadOwnTrip(w, r, driverID)
	if !ok {
		return
	}
	if trip.Status != store.TripStatusCreated && trip.Status != store.TripStatusInProgress {
		writeError(w, http.StatusConflict, "trip is not currently active")
		return
	}

	var req positionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	s.WS.ReportPosition(trip, req.Lat, req.Lon)
	w.WriteHeader(http.StatusNoContent)
}

// handleCompleteTrip lets the driver explicitly confirm arrival - real GPS
// tracking has no reliable auto-arrival signal on its own.
func (s *Server) handleCompleteTrip(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	trip, ok := s.loadOwnTrip(w, r, driverID)
	if !ok {
		return
	}
	if trip.Status != store.TripStatusCreated && trip.Status != store.TripStatusInProgress {
		writeError(w, http.StatusConflict, "trip is not currently active")
		return
	}

	if err := s.Trips.MarkCompleted(r.Context(), trip.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to complete trip: "+err.Error())
		return
	}
	if _, err := s.TripEvents.Create(r.Context(), trip.ID, "arrived", "Arrived at destination"); err != nil {
		log.Printf("log arrived event for trip %d: %v", trip.ID, err)
	}
	s.WS.CompleteTrip(trip.ID)

	trip.Status = store.TripStatusCompleted
	writeJSON(w, http.StatusOK, toTripResponse(trip, nil))
}

// handleListTrips branches by caller role: a driver sees their own trips
// (self-service and assigned alike); a dispatcher sees the trips they've
// assigned. Optional ?status= filter.
func (s *Server) handleListTrips(w http.ResponseWriter, r *http.Request) {
	callerID, _ := driverIDFromContext(r.Context())

	caller, err := s.loadAccount(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account: "+err.Error())
		return
	}

	var status *string
	if v := r.URL.Query().Get("status"); v != "" {
		status = &v
	}

	isDispatcher := caller.Role == store.RoleDispatcher
	trips, err := s.Trips.ListForOwner(r.Context(), callerID, isDispatcher, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list trips: "+err.Error())
		return
	}

	// Driver usernames are only useful (and only fetched) for the dispatcher's
	// own trip list - a driver already knows these are their own trips.
	usernames := map[int64]string{}
	out := make([]tripResponse, len(trips))
	for i, t := range trips {
		resp := toTripResponse(t, nil)
		if isDispatcher {
			if name, ok := usernames[t.DriverID]; ok {
				resp.DriverUsername = name
			} else if driver, err := s.Drivers.Get(r.Context(), t.DriverID); err == nil {
				usernames[t.DriverID] = driver.Username
				resp.DriverUsername = driver.Username
			}
		}
		out[i] = resp
	}
	writeJSON(w, http.StatusOK, out)
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
	if !tripAccessible(trip, driverID) {
		writeError(w, http.StatusForbidden, "trip does not belong to you")
		return
	}

	writeJSON(w, http.StatusOK, toTripResponse(trip, nil))
}

// handleTripStream checks trip ownership (the ws package itself knows nothing
// about auth) before delegating to the WebSocket gateway. Accessible to the
// assigned driver AND the dispatcher who assigned the trip (live tracking).
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
	if !tripAccessible(trip, driverID) {
		writeError(w, http.StatusForbidden, "trip does not belong to you")
		return
	}

	s.WS.HandleTripStream(w, r)
}

type tripEventResponse struct {
	ID          int64     `json:"id"`
	EventType   string    `json:"event_type"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
}

func (s *Server) handleListTripEvents(w http.ResponseWriter, r *http.Request) {
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
	if !tripAccessible(trip, driverID) {
		writeError(w, http.StatusForbidden, "trip does not belong to you")
		return
	}

	events, err := s.TripEvents.ListByTrip(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list trip events: "+err.Error())
		return
	}

	out := make([]tripEventResponse, len(events))
	for i, e := range events {
		out[i] = tripEventResponse{ID: e.ID, EventType: e.EventType, Description: e.Description, OccurredAt: e.OccurredAt}
	}
	writeJSON(w, http.StatusOK, out)
}
