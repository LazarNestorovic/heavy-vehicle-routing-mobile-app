package queue

import (
	"fmt"
	"time"
)

const RoutingKeyTripStarted = "trip.started"

type TripStartedEvent struct {
	TripID int64 `json:"trip_id"`
}

// ChatRoutingKey is the same regardless of who's sending, so a single binding
// covers a conversation in both directions.
func ChatRoutingKey(a, b int64) string {
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("chat.%d.%d", a, b)
}

type ChatMessageEvent struct {
	ID           int64     `json:"id"`
	FromDriverID int64     `json:"from_driver_id"`
	ToDriverID   int64     `json:"to_driver_id"`
	Body         string    `json:"body"`
	SentAt       time.Time `json:"sent_at"`
}
