package store

import (
	"context"
	"database/sql"
	"fmt"
)

type FavoriteStop struct {
	ID       int64
	DriverID int64
	Lat      float64
	Lon      float64
	Name     string
}

type FavoriteStopStore struct {
	db *sql.DB
}

func NewFavoriteStopStore(db *sql.DB) *FavoriteStopStore {
	return &FavoriteStopStore{db: db}
}

func (s *FavoriteStopStore) Create(ctx context.Context, f FavoriteStop) (FavoriteStop, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO driver_favorite_stops (driver_id, lat, lon, name)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, f.DriverID, f.Lat, f.Lon, f.Name)

	if err := row.Scan(&f.ID); err != nil {
		return FavoriteStop{}, fmt.Errorf("insert favorite stop: %w", err)
	}
	return f, nil
}

func (s *FavoriteStopStore) List(ctx context.Context, driverID int64) ([]FavoriteStop, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, lat, lon, name FROM driver_favorite_stops WHERE driver_id = $1 ORDER BY id DESC`, driverID)
	if err != nil {
		return nil, fmt.Errorf("list favorite stops: %w", err)
	}
	defer rows.Close()

	stops := []FavoriteStop{}
	for rows.Next() {
		f := FavoriteStop{DriverID: driverID}
		if err := rows.Scan(&f.ID, &f.Lat, &f.Lon, &f.Name); err != nil {
			return nil, fmt.Errorf("scan favorite stop: %w", err)
		}
		stops = append(stops, f)
	}
	return stops, rows.Err()
}

// Delete removes a favorite stop, scoped to driverID so one driver can't delete
// another's. Returns ErrNotFound if nothing matched - covers "doesn't exist" and
// "not yours" the same way, without leaking which one it was.
func (s *FavoriteStopStore) Delete(ctx context.Context, id, driverID int64) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM driver_favorite_stops WHERE id = $1 AND driver_id = $2`, id, driverID)
	if err != nil {
		return fmt.Errorf("delete favorite stop: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete favorite stop: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
