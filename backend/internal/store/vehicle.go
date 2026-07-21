package store

import (
	"context"
	"database/sql"
	"fmt"
)

type Vehicle struct {
	ID         int64
	HeightM    float64
	WidthM     float64
	LengthM    float64
	WeightKg   float64
	AxleLoadKg float64
	Hazmat     bool
}

type VehicleStore struct {
	db *sql.DB
}

func NewVehicleStore(db *sql.DB) *VehicleStore {
	return &VehicleStore{db: db}
}

func (s *VehicleStore) Create(ctx context.Context, v Vehicle) (Vehicle, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO vehicles (height_m, width_m, length_m, weight_kg, axle_load_kg, hazmat)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		v.HeightM, v.WidthM, v.LengthM, v.WeightKg, v.AxleLoadKg, v.Hazmat)

	if err := row.Scan(&v.ID); err != nil {
		return Vehicle{}, fmt.Errorf("insert vehicle: %w", err)
	}
	return v, nil
}

var ErrNotFound = fmt.Errorf("not found")

func (s *VehicleStore) Get(ctx context.Context, id int64) (Vehicle, error) {
	var v Vehicle
	v.ID = id
	row := s.db.QueryRowContext(ctx, `
		SELECT height_m, width_m, length_m, weight_kg, axle_load_kg, hazmat
		FROM vehicles WHERE id = $1`, id)

	if err := row.Scan(&v.HeightM, &v.WidthM, &v.LengthM, &v.WeightKg, &v.AxleLoadKg, &v.Hazmat); err != nil {
		if err == sql.ErrNoRows {
			return Vehicle{}, ErrNotFound
		}
		return Vehicle{}, fmt.Errorf("select vehicle: %w", err)
	}
	return v, nil
}
