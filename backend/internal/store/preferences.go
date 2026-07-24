package store

import (
	"context"
	"database/sql"
	"fmt"
)

// DriverPreferences drives the dynamic scoring formula in internal/scoring.
// Priorities are 1-5; 3 is "neutral" (close to the original fixed-weight formula).
type DriverPreferences struct {
	DriverID           int64
	FuelPriority       int
	CargoPriority      int
	HighwayPriority    int
	TimePriority       int
	PreferredFuelBrand *string
}

func defaultPreferences(driverID int64) DriverPreferences {
	return DriverPreferences{
		DriverID: driverID, FuelPriority: 3, CargoPriority: 3, HighwayPriority: 3, TimePriority: 3,
	}
}

type PreferencesStore struct {
	db *sql.DB
}

func NewPreferencesStore(db *sql.DB) *PreferencesStore {
	return &PreferencesStore{db: db}
}

// Get returns the driver's saved preferences, or sensible defaults if they've
// never set any (no row yet) - absence is not an error here, unlike other Get methods.
func (s *PreferencesStore) Get(ctx context.Context, driverID int64) (DriverPreferences, error) {
	var p DriverPreferences
	p.DriverID = driverID
	row := s.db.QueryRowContext(ctx, `
		SELECT fuel_priority, cargo_priority, highway_priority, time_priority, preferred_fuel_brand
		FROM driver_preferences WHERE driver_id = $1`, driverID)

	if err := row.Scan(&p.FuelPriority, &p.CargoPriority, &p.HighwayPriority, &p.TimePriority, &p.PreferredFuelBrand); err != nil {
		if err == sql.ErrNoRows {
			return defaultPreferences(driverID), nil
		}
		return DriverPreferences{}, fmt.Errorf("select preferences: %w", err)
	}
	return p, nil
}

func (s *PreferencesStore) Upsert(ctx context.Context, p DriverPreferences) (DriverPreferences, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO driver_preferences (driver_id, fuel_priority, cargo_priority, highway_priority, time_priority, preferred_fuel_brand)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (driver_id) DO UPDATE SET
			fuel_priority = EXCLUDED.fuel_priority,
			cargo_priority = EXCLUDED.cargo_priority,
			highway_priority = EXCLUDED.highway_priority,
			time_priority = EXCLUDED.time_priority,
			preferred_fuel_brand = EXCLUDED.preferred_fuel_brand`,
		p.DriverID, p.FuelPriority, p.CargoPriority, p.HighwayPriority, p.TimePriority, p.PreferredFuelBrand)
	if err != nil {
		return DriverPreferences{}, fmt.Errorf("upsert preferences: %w", err)
	}
	return p, nil
}
