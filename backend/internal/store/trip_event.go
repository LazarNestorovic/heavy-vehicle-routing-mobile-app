package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type TripEvent struct {
	ID          int64
	TripID      int64
	EventType   string // "departed" | "rest_stop_suggested" | "arrived"
	Description string
	OccurredAt  time.Time
}

type TripEventStore struct {
	db *sql.DB
}

func NewTripEventStore(db *sql.DB) *TripEventStore {
	return &TripEventStore{db: db}
}

func (s *TripEventStore) Create(ctx context.Context, tripID int64, eventType, description string) (TripEvent, error) {
	e := TripEvent{TripID: tripID, EventType: eventType, Description: description}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO trip_events (trip_id, event_type, description)
		VALUES ($1, $2, $3)
		RETURNING id, occurred_at`, tripID, eventType, description)

	if err := row.Scan(&e.ID, &e.OccurredAt); err != nil {
		return TripEvent{}, fmt.Errorf("insert trip event: %w", err)
	}
	return e, nil
}

func (s *TripEventStore) ListByTrip(ctx context.Context, tripID int64) ([]TripEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_type, description, occurred_at FROM trip_events
		WHERE trip_id = $1 ORDER BY occurred_at ASC`, tripID)
	if err != nil {
		return nil, fmt.Errorf("list trip events: %w", err)
	}
	defer rows.Close()

	events := []TripEvent{}
	for rows.Next() {
		e := TripEvent{TripID: tripID}
		if err := rows.Scan(&e.ID, &e.EventType, &e.Description, &e.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan trip event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
