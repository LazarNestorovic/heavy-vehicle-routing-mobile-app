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
//     immediately (status "created", departed event fires now).
//   - Managed driver (has a dispatcher): rejected - their dispatcher creates trips
//     for them.
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
		writeError(w, http.StatusForbidden, "your dispatcher creates trips for you")
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
	ranked, err := s.bestRoute(r.Context(), req.Origin, req.Destination, profile, prefs, preferredStops)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	best := ranked[0]
	explanation := s.Explain.Explain(r.Context(), req.Origin, req.Destination, toTruckProfile(profile), best.RouteCandidate, prefs, profile.WeightKg, preferredStops)

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

	if err := s.Trips.MarkStarted(r.Context(), trip.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start trip: "+err.Error())
		return
	}
	s.startTripSideEffects(r.Context(), trip.ID)

	trip.Status = store.TripStatusCreated
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
