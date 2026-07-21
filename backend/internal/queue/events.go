package queue

const (
	RoutingKeyTripStarted    = "trip.started"
	RoutingKeyTripETAUpdated = "trip.eta_updated"
)

type TripStartedEvent struct {
	TripID int64 `json:"trip_id"`
}

type TripETAUpdatedEvent struct {
	TripID                int64    `json:"trip_id"`
	DurationMin           float64  `json:"duration_min"`
	NextRestSuggestionMin *float64 `json:"next_rest_suggestion_min,omitempty"`
}
