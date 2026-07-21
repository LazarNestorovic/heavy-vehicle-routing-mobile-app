package store

import (
	"context"
	"database/sql"
	"fmt"
)

type Trip struct {
	ID                    int64
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
	NextRestSuggestionMin *float64
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
		INSERT INTO trips (vehicle_id, origin_lat, origin_lon, destination_lat, destination_lon,
			distance_km, duration_min, risk_score, shape, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		t.VehicleID, t.OriginLat, t.OriginLon, t.DestinationLat, t.DestinationLon,
		t.DistanceKm, t.DurationMin, t.RiskScore, t.Shape, t.Status)

	if err := row.Scan(&t.ID); err != nil {
		return Trip{}, fmt.Errorf("insert trip: %w", err)
	}
	return t, nil
}

func (s *TripStore) Get(ctx context.Context, id int64) (Trip, error) {
	var t Trip
	t.ID = id
	row := s.db.QueryRowContext(ctx, `
		SELECT vehicle_id, origin_lat, origin_lon, destination_lat, destination_lon,
			distance_km, duration_min, risk_score, shape, status, next_rest_suggestion_min
		FROM trips WHERE id = $1`, id)

	if err := row.Scan(&t.VehicleID, &t.OriginLat, &t.OriginLon, &t.DestinationLat, &t.DestinationLon,
		&t.DistanceKm, &t.DurationMin, &t.RiskScore, &t.Shape, &t.Status, &t.NextRestSuggestionMin); err != nil {
		if err == sql.ErrNoRows {
			return Trip{}, ErrNotFound
		}
		return Trip{}, fmt.Errorf("select trip: %w", err)
	}
	return t, nil
}

// UpdateAfterProcessing is called by the trip.started worker once it has computed a
// rest-stop suggestion, moving the trip to "in_progress".
func (s *TripStore) UpdateAfterProcessing(ctx context.Context, id int64, nextRestSuggestionMin *float64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE trips SET status = 'in_progress', next_rest_suggestion_min = $2 WHERE id = $1`,
		id, nextRestSuggestionMin)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	return nil
}
