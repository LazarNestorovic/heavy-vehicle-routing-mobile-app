package store

import (
	"context"
	"database/sql"
	"fmt"
)

// Vehicle belongs to EITHER a driver (personal) OR a dispatcher (fleet) -
// exactly one of DriverID/DispatcherID is set. That "exactly one" rule is
// enforced at the Go handler level, not a DB CHECK (see documentations/
// features/ entry for the dispatcher/driver roles feature).
type Vehicle struct {
	ID            int64
	DriverID      *int64
	DispatcherID  *int64
	HeightM       float64
	WidthM        float64
	LengthM       float64
	WeightKg      float64
	AxleLoadKg    float64
	Hazmat        bool
	FuelPercent   float64
	NextServiceKm *float64
}

type VehicleStore struct {
	db *sql.DB
}

func NewVehicleStore(db *sql.DB) *VehicleStore {
	return &VehicleStore{db: db}
}

// Create only sets the vehicle's physical dimensions - fuel_percent/next_service_km
// are status fields updated later via UpdateStatus, so they're left to their DB
// defaults here (fuel_percent=100, next_service_km=NULL) rather than risking a
// Go zero-value (0) being inserted for a field the caller didn't think to set.
// Exactly one of v.DriverID/v.DispatcherID must be set by the caller.
func (s *VehicleStore) Create(ctx context.Context, v Vehicle) (Vehicle, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO vehicles (driver_id, dispatcher_id, height_m, width_m, length_m, weight_kg, axle_load_kg, hazmat)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, fuel_percent, next_service_km`,
		v.DriverID, v.DispatcherID, v.HeightM, v.WidthM, v.LengthM, v.WeightKg, v.AxleLoadKg, v.Hazmat)

	if err := row.Scan(&v.ID, &v.FuelPercent, &v.NextServiceKm); err != nil {
		return Vehicle{}, fmt.Errorf("insert vehicle: %w", err)
	}
	return v, nil
}

var ErrNotFound = fmt.Errorf("not found")

func (s *VehicleStore) Get(ctx context.Context, id int64) (Vehicle, error) {
	var v Vehicle
	v.ID = id
	row := s.db.QueryRowContext(ctx, `
		SELECT driver_id, dispatcher_id, height_m, width_m, length_m, weight_kg, axle_load_kg, hazmat, fuel_percent, next_service_km
		FROM vehicles WHERE id = $1`, id)

	if err := row.Scan(&v.DriverID, &v.DispatcherID, &v.HeightM, &v.WidthM, &v.LengthM, &v.WeightKg, &v.AxleLoadKg, &v.Hazmat, &v.FuelPercent, &v.NextServiceKm); err != nil {
		if err == sql.ErrNoRows {
			return Vehicle{}, ErrNotFound
		}
		return Vehicle{}, fmt.Errorf("select vehicle: %w", err)
	}
	return v, nil
}

// List returns every personal vehicle owned by driverID, most recently created first.
func (s *VehicleStore) List(ctx context.Context, driverID int64) ([]Vehicle, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, driver_id, dispatcher_id, height_m, width_m, length_m, weight_kg, axle_load_kg, hazmat, fuel_percent, next_service_km
		FROM vehicles WHERE driver_id = $1 ORDER BY id DESC`, driverID)
	if err != nil {
		return nil, fmt.Errorf("list vehicles: %w", err)
	}
	defer rows.Close()

	vehicles := []Vehicle{}
	for rows.Next() {
		var v Vehicle
		if err := rows.Scan(&v.ID, &v.DriverID, &v.DispatcherID, &v.HeightM, &v.WidthM, &v.LengthM, &v.WeightKg, &v.AxleLoadKg, &v.Hazmat, &v.FuelPercent, &v.NextServiceKm); err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, rows.Err()
}

// ListFleet returns every fleet vehicle owned by dispatcherID, most recently created first.
func (s *VehicleStore) ListFleet(ctx context.Context, dispatcherID int64) ([]Vehicle, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, driver_id, dispatcher_id, height_m, width_m, length_m, weight_kg, axle_load_kg, hazmat, fuel_percent, next_service_km
		FROM vehicles WHERE dispatcher_id = $1 ORDER BY id DESC`, dispatcherID)
	if err != nil {
		return nil, fmt.Errorf("list fleet vehicles: %w", err)
	}
	defer rows.Close()

	vehicles := []Vehicle{}
	for rows.Next() {
		var v Vehicle
		if err := rows.Scan(&v.ID, &v.DriverID, &v.DispatcherID, &v.HeightM, &v.WidthM, &v.LengthM, &v.WeightKg, &v.AxleLoadKg, &v.Hazmat, &v.FuelPercent, &v.NextServiceKm); err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, rows.Err()
}

// UpdateStatus sets the manually-reported fuel/service fields (see comment on
// the schema migration in internal/db/db.go - no telematics integration).
func (s *VehicleStore) UpdateStatus(ctx context.Context, id int64, fuelPercent float64, nextServiceKm *float64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE vehicles SET fuel_percent = $2, next_service_km = $3 WHERE id = $1`,
		id, fuelPercent, nextServiceKm)
	if err != nil {
		return fmt.Errorf("update vehicle status: %w", err)
	}
	return nil
}
