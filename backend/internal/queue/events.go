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
	RestStopLat           *float64 `json:"rest_stop_lat,omitempty"`
	RestStopLon           *float64 `json:"rest_stop_lon,omitempty"`
	RestStopName          *string  `json:"rest_stop_name,omitempty"`
	RestStopAmenity       *string  `json:"rest_stop_amenity,omitempty"`
}
