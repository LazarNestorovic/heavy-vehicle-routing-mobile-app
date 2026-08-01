package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Trip struct {
	ID                    int64
	DriverID              int64
	AssignedByID          *int64 // set when a dispatcher created/assigned this trip; NULL for self-service
	VehicleID             int64
	OriginLat             float64
	OriginLon             float64
	DestinationLat        float64
	DestinationLon        float64
	DistanceKm            float64
	DurationMin           float64
	RiskScore             float64
	Shape                 string
	Status                string
	Explanation           *string
	NextRestSuggestionMin *float64
	RestStopLat           *float64
	RestStopLon           *float64
	RestStopName          *string
	RestStopAmenity       *string
	CargoDescription      *string
	CargoWeightKg         *float64
	CargoTempRange        *string
	PickupLocation        *string
	DropoffLocation       *string
	CreatedAt             time.Time
}

// Status machine for a dispatcher-assigned trip: offered -> accepted -> created
// (started) -> in_progress (worker picks it up). "rejected" is a dead end -
// the driver declined it. Self-service trips (no dispatcher) skip straight to
// "created" at creation time, same as before this state machine existed.
const (
	TripStatusOffered    = "offered"
	TripStatusAccepted   = "accepted"
	TripStatusRejected   = "rejected"
	TripStatusCreated    = "created"
	TripStatusInProgress = "in_progress"
	TripStatusCompleted  = "completed"
)

// RestStopSuggestion is what the trip.started worker computes and persists.
type RestStopSuggestion struct {
	AfterMinutes *float64
	Lat          *float64
	Lon          *float64
	Name         *string
	Amenity      *string
}

// DrivingHours is a simplified stand-in for real AETR driving-hours tracking -
// see documentations/features/2026-07-21-nocturne-redesign.md for the honest
// scope note (no cumulative multi-day HOS ledger, just derived from this
// vehicle's own trips/trip_events).
type DrivingHours struct {
	SinceLastBreakMin float64
	DrivingTodayMin   float64
}

type TripStore struct {
	db *sql.DB
}

func NewTripStore(db *sql.DB) *TripStore {
	return &TripStore{db: db}
}

func (s *TripStore) Create(ctx context.Context, t Trip) (Trip, error) {
	if t.Status == "" {
		t.Status = "created"
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO trips (driver_id, assigned_by_id, vehicle_id, origin_lat, origin_lon, destination_lat, destination_lon,
			distance_km, duration_min, risk_score, shape, status, explanation,
			cargo_description, cargo_weight_kg, cargo_temp_range, pickup_location, dropoff_location)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, created_at`,
		t.DriverID, t.AssignedByID, t.VehicleID, t.OriginLat, t.OriginLon, t.DestinationLat, t.DestinationLon,
		t.DistanceKm, t.DurationMin, t.RiskScore, t.Shape, t.Status, t.Explanation,
		t.CargoDescription, t.CargoWeightKg, t.CargoTempRange, t.PickupLocation, t.DropoffLocation)

	if err := row.Scan(&t.ID, &t.CreatedAt); err != nil {
		return Trip{}, fmt.Errorf("insert trip: %w", err)
	}
	return t, nil
}

func (s *TripStore) Get(ctx context.Context, id int64) (Trip, error) {
	var t Trip
	t.ID = id
	row := s.db.QueryRowContext(ctx, `
		SELECT driver_id, assigned_by_id, vehicle_id, origin_lat, origin_lon, destination_lat, destination_lon,
			distance_km, duration_min, risk_score, shape, status, explanation,
			next_rest_suggestion_min, rest_stop_lat, rest_stop_lon, rest_stop_name, rest_stop_amenity,
			cargo_description, cargo_weight_kg, cargo_temp_range, pickup_location, dropoff_location, created_at
		FROM trips WHERE id = $1`, id)

	if err := row.Scan(&t.DriverID, &t.AssignedByID, &t.VehicleID, &t.OriginLat, &t.OriginLon, &t.DestinationLat, &t.DestinationLon,
		&t.DistanceKm, &t.DurationMin, &t.RiskScore, &t.Shape, &t.Status, &t.Explanation,
		&t.NextRestSuggestionMin, &t.RestStopLat, &t.RestStopLon, &t.RestStopName, &t.RestStopAmenity,
		&t.CargoDescription, &t.CargoWeightKg, &t.CargoTempRange, &t.PickupLocation, &t.DropoffLocation, &t.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return Trip{}, ErrNotFound
		}
		return Trip{}, fmt.Errorf("select trip: %w", err)
	}
	return t, nil
}

// markStatusTransition is the shared implementation behind MarkAccepted/
// MarkRejected/MarkStarted: update status only if it's currently `from`,
// returning ErrNotFound if the trip doesn't exist or isn't in that state
// (the handler is responsible for telling those two cases apart via a
// preceding Get, same pattern as the rest of this package).
func (s *TripStore) markStatusTransition(ctx context.Context, id int64, from, to string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE trips SET status = $2 WHERE id = $1 AND status = $3`,
		id, to, from)
	if err != nil {
		return fmt.Errorf("update trip status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update trip status: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAccepted transitions a dispatcher-assigned trip from 'offered' to
// 'accepted' (the driver reviewed it - route/cargo/vehicle - and committed to
// it, but hasn't departed yet).
func (s *TripStore) MarkAccepted(ctx context.Context, id int64) error {
	return s.markStatusTransition(ctx, id, TripStatusOffered, TripStatusAccepted)
}

// MarkRejected transitions a dispatcher-assigned trip from 'offered' to
// 'rejected' (the driver declined it).
func (s *TripStore) MarkRejected(ctx context.Context, id int64) error {
	return s.markStatusTransition(ctx, id, TripStatusOffered, TripStatusRejected)
}

// MarkStarted transitions a dispatcher-assigned trip from 'accepted' to
// 'created' (the driver actually departs) - see httpapi handleStartTrip.
func (s *TripStore) MarkStarted(ctx context.Context, id int64) error {
	return s.markStatusTransition(ctx, id, TripStatusAccepted, TripStatusCreated)
}

// MarkCompleted transitions an active trip to 'completed' - the driver
// explicitly confirmed arrival (see httpapi handleCompleteTrip; live GPS has
// no reliable auto-arrival signal on its own). Tries 'in_progress' first (the common case - the
// trip.started worker has already run) and falls back to 'created' (a real
// GPS ping/complete can in principle race ahead of the worker).
func (s *TripStore) MarkCompleted(ctx context.Context, id int64) error {
	err := s.markStatusTransition(ctx, id, TripStatusInProgress, TripStatusCompleted)
	if err == ErrNotFound {
		err = s.markStatusTransition(ctx, id, TripStatusCreated, TripStatusCompleted)
	}
	return err
}

// HasActiveTrip returns driverID's currently active trip (status "created" or
// "in_progress"), or nil if they have none - used to block starting a second
// trip while one is already underway (see httpapi handleCreateTrip/
// handleStartTrip), and by the app to offer "resume" instead of planning a
// new route.
func (s *TripStore) HasActiveTrip(ctx context.Context, driverID int64) (*Trip, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, driver_id, assigned_by_id, vehicle_id, origin_lat, origin_lon, destination_lat, destination_lon,
			distance_km, duration_min, risk_score, shape, status, explanation,
			next_rest_suggestion_min, rest_stop_lat, rest_stop_lon, rest_stop_name, rest_stop_amenity,
			cargo_description, cargo_weight_kg, cargo_temp_range, pickup_location, dropoff_location, created_at
		FROM trips
		WHERE driver_id = $1 AND status IN ($2, $3)
		ORDER BY created_at DESC
		LIMIT 1`, driverID, TripStatusCreated, TripStatusInProgress)

	var t Trip
	err := row.Scan(&t.ID, &t.DriverID, &t.AssignedByID, &t.VehicleID, &t.OriginLat, &t.OriginLon, &t.DestinationLat, &t.DestinationLon,
		&t.DistanceKm, &t.DurationMin, &t.RiskScore, &t.Shape, &t.Status, &t.Explanation,
		&t.NextRestSuggestionMin, &t.RestStopLat, &t.RestStopLon, &t.RestStopName, &t.RestStopAmenity,
		&t.CargoDescription, &t.CargoWeightKg, &t.CargoTempRange, &t.PickupLocation, &t.DropoffLocation, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("check active trip: %w", err)
	}
	return &t, nil
}

// HasPendingOffer returns driverID's pending dispatcher-assigned trip
// (status "offered" or "accepted" - requested but not yet started), or nil
// if they have none. Offered/accepted only ever occurs for dispatcher-
// assigned trips (self-service ones skip straight to "created"), so no
// separate assigned_by_id filter is needed. Used to stop a managed driver
// from self-planning a route on their own vehicle while a request from
// their dispatcher is still awaiting a decision or accepted but not yet
// under way (see httpapi handleCreateTrip).
func (s *TripStore) HasPendingOffer(ctx context.Context, driverID int64) (*Trip, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, driver_id, assigned_by_id, vehicle_id, origin_lat, origin_lon, destination_lat, destination_lon,
			distance_km, duration_min, risk_score, shape, status, explanation,
			next_rest_suggestion_min, rest_stop_lat, rest_stop_lon, rest_stop_name, rest_stop_amenity,
			cargo_description, cargo_weight_kg, cargo_temp_range, pickup_location, dropoff_location, created_at
		FROM trips
		WHERE driver_id = $1 AND status IN ($2, $3)
		ORDER BY created_at DESC
		LIMIT 1`, driverID, TripStatusOffered, TripStatusAccepted)

	var t Trip
	err := row.Scan(&t.ID, &t.DriverID, &t.AssignedByID, &t.VehicleID, &t.OriginLat, &t.OriginLon, &t.DestinationLat, &t.DestinationLon,
		&t.DistanceKm, &t.DurationMin, &t.RiskScore, &t.Shape, &t.Status, &t.Explanation,
		&t.NextRestSuggestionMin, &t.RestStopLat, &t.RestStopLon, &t.RestStopName, &t.RestStopAmenity,
		&t.CargoDescription, &t.CargoWeightKg, &t.CargoTempRange, &t.PickupLocation, &t.DropoffLocation, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("check pending offer: %w", err)
	}
	return &t, nil
}

// EditByDispatcher updates a dispatcher-assigned trip's vehicle/route/cargo
// while it's still pending the driver's decision ("offered") or accepted but
// not yet under way ("accepted") - see httpapi handleUpdateTrip. Always sets
// status to "offered": an edit to an already-"offered" trip is a no-op on
// status, but an edit to an "accepted" one reverts it - the trip changed
// after the driver committed to it, so they need to see the new version and
// accept/reject again. Clears any rest-stop suggestion, same reasoning as
// Reroute(). Guarded by WHERE status IN (...) so a concurrent accept/start
// can't be silently overwritten; returns ErrNotFound if the trip no longer
// matches (deleted, or moved past both of those statuses in the meantime).
func (s *TripStore) EditByDispatcher(ctx context.Context, id int64, t Trip) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE trips SET vehicle_id = $2, origin_lat = $3, origin_lon = $4,
			destination_lat = $5, destination_lon = $6, distance_km = $7, duration_min = $8,
			risk_score = $9, shape = $10, explanation = $11,
			cargo_description = $12, cargo_weight_kg = $13, cargo_temp_range = $14,
			pickup_location = $15, dropoff_location = $16, status = $17,
			next_rest_suggestion_min = NULL, rest_stop_lat = NULL, rest_stop_lon = NULL,
			rest_stop_name = NULL, rest_stop_amenity = NULL
		WHERE id = $1 AND status IN ($17, $18)`,
		id, t.VehicleID, t.OriginLat, t.OriginLon, t.DestinationLat, t.DestinationLon,
		t.DistanceKm, t.DurationMin, t.RiskScore, t.Shape, t.Explanation,
		t.CargoDescription, t.CargoWeightKg, t.CargoTempRange, t.PickupLocation, t.DropoffLocation,
		TripStatusOffered, TripStatusAccepted)
	if err != nil {
		return fmt.Errorf("edit trip: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("edit trip: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListForOwner lists trips belonging to ownerID. byAssigner=false filters by
// driver_id (a driver's own trips - self-service and assigned alike);
// byAssigner=true filters by assigned_by_id (a dispatcher's assigned trips).
// status, if non-nil, additionally filters by trip status.
func (s *TripStore) ListForOwner(ctx context.Context, ownerID int64, byAssigner bool, status *string) ([]Trip, error) {
	column := "driver_id"
	if byAssigner {
		column = "assigned_by_id"
	}
	query := `
		SELECT id, driver_id, assigned_by_id, vehicle_id, origin_lat, origin_lon, destination_lat, destination_lon,
			distance_km, duration_min, risk_score, shape, status, explanation,
			next_rest_suggestion_min, rest_stop_lat, rest_stop_lon, rest_stop_name, rest_stop_amenity,
			cargo_description, cargo_weight_kg, cargo_temp_range, pickup_location, dropoff_location, created_at
		FROM trips WHERE ` + column + ` = $1`
	args := []any{ownerID}
	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list trips for owner: %w", err)
	}
	defer rows.Close()

	trips := []Trip{}
	for rows.Next() {
		var t Trip
		if err := rows.Scan(&t.ID, &t.DriverID, &t.AssignedByID, &t.VehicleID, &t.OriginLat, &t.OriginLon, &t.DestinationLat, &t.DestinationLon,
			&t.DistanceKm, &t.DurationMin, &t.RiskScore, &t.Shape, &t.Status, &t.Explanation,
			&t.NextRestSuggestionMin, &t.RestStopLat, &t.RestStopLon, &t.RestStopName, &t.RestStopAmenity,
			&t.CargoDescription, &t.CargoWeightKg, &t.CargoTempRange, &t.PickupLocation, &t.DropoffLocation, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan trip: %w", err)
		}
		trips = append(trips, t)
	}
	return trips, rows.Err()
}

// Reroute updates a trip's origin and route after the driver deviated from
// the planned path and accepted a recalculated one (see httpapi
// handleRerouteTrip) - the destination is unchanged. Clears any rest-stop
// suggestion tied to the OLD route (computed for a path that may no longer be
// driven at all) - a fresh one isn't recomputed here; the caller re-publishes
// trip.started so the existing worker does that, the same way it does for a
// brand new trip.
func (s *TripStore) Reroute(ctx context.Context, id int64, originLat, originLon, distanceKm, durationMin, riskScore float64, shape string, explanation *string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE trips SET origin_lat = $2, origin_lon = $3, distance_km = $4, duration_min = $5,
			risk_score = $6, shape = $7, explanation = $8,
			next_rest_suggestion_min = NULL, rest_stop_lat = NULL, rest_stop_lon = NULL,
			rest_stop_name = NULL, rest_stop_amenity = NULL
		WHERE id = $1`,
		id, originLat, originLon, distanceKm, durationMin, riskScore, shape, explanation)
	if err != nil {
		return fmt.Errorf("reroute trip: %w", err)
	}
	return nil
}

// UpdateAfterProcessing is called by the trip.started worker once it has computed a
// rest-stop suggestion (if any), moving the trip to "in_progress".
func (s *TripStore) UpdateAfterProcessing(ctx context.Context, id int64, rest RestStopSuggestion) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE trips SET status = 'in_progress', next_rest_suggestion_min = $2,
			rest_stop_lat = $3, rest_stop_lon = $4, rest_stop_name = $5, rest_stop_amenity = $6
		WHERE id = $1`,
		id, rest.AfterMinutes, rest.Lat, rest.Lon, rest.Name, rest.Amenity)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	return nil
}

// DrivingHours computes SinceLastBreakMin (minutes since the vehicle's most
// recent 'departed' or 'rest_stop_suggested' trip_event) and DrivingTodayMin
// (summed duration_min of the vehicle's trips created today).
func (s *TripStore) DrivingHours(ctx context.Context, vehicleID int64) (DrivingHours, error) {
	var h DrivingHours

	var lastBreak sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(te.occurred_at) FROM trip_events te
		JOIN trips t ON t.id = te.trip_id
		WHERE t.vehicle_id = $1 AND te.event_type IN ('departed', 'rest_stop_suggested')`,
		vehicleID).Scan(&lastBreak)
	if err != nil {
		return DrivingHours{}, fmt.Errorf("query last break: %w", err)
	}
	if lastBreak.Valid {
		h.SinceLastBreakMin = time.Since(lastBreak.Time).Minutes()
	}

	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(duration_min), 0) FROM trips
		WHERE vehicle_id = $1 AND created_at::date = CURRENT_DATE`,
		vehicleID).Scan(&h.DrivingTodayMin)
	if err != nil {
		return DrivingHours{}, fmt.Errorf("query driving today: %w", err)
	}

	return h, nil
}
