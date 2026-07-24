package store

import (
	"context"
	"database/sql"
	"fmt"
)

type Trip struct {
	ID                    int64
	DriverID              int64
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
}

// RestStopSuggestion is what the trip.started worker computes and persists.
type RestStopSuggestion struct {
	AfterMinutes *float64
	Lat          *float64
	Lon          *float64
	Name         *string
	Amenity      *string
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
		INSERT INTO trips (driver_id, vehicle_id, origin_lat, origin_lon, destination_lat, destination_lon,
			distance_km, duration_min, risk_score, shape, status, explanation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		t.DriverID, t.VehicleID, t.OriginLat, t.OriginLon, t.DestinationLat, t.DestinationLon,
		t.DistanceKm, t.DurationMin, t.RiskScore, t.Shape, t.Status, t.Explanation)

	if err := row.Scan(&t.ID); err != nil {
		return Trip{}, fmt.Errorf("insert trip: %w", err)
	}
	return t, nil
}

func (s *TripStore) Get(ctx context.Context, id int64) (Trip, error) {
	var t Trip
	t.ID = id
	row := s.db.QueryRowContext(ctx, `
		SELECT driver_id, vehicle_id, origin_lat, origin_lon, destination_lat, destination_lon,
			distance_km, duration_min, risk_score, shape, status, explanation,
			next_rest_suggestion_min, rest_stop_lat, rest_stop_lon, rest_stop_name, rest_stop_amenity
		FROM trips WHERE id = $1`, id)

	if err := row.Scan(&t.DriverID, &t.VehicleID, &t.OriginLat, &t.OriginLon, &t.DestinationLat, &t.DestinationLon,
		&t.DistanceKm, &t.DurationMin, &t.RiskScore, &t.Shape, &t.Status, &t.Explanation,
		&t.NextRestSuggestionMin, &t.RestStopLat, &t.RestStopLon, &t.RestStopName, &t.RestStopAmenity); err != nil {
		if err == sql.ErrNoRows {
			return Trip{}, ErrNotFound
		}
		return Trip{}, fmt.Errorf("select trip: %w", err)
	}
	return t, nil
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
