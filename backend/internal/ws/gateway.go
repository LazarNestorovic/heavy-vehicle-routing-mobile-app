// Package ws is the WebSocket gateway from SPECIFIKACIJA.md 3.7: pushes a
// simulated GPS position along a trip's route to a connected client. There's no
// real vehicle/phone in this project (see the guide's timeline constraints), so
// position is simulated by walking the route's decoded shape at a fixed
// wall-clock pace - a standard, accepted substitute for live GPS in a thesis
// demo (documentations/features/2026-07-21-websocket-gateway.md).
package ws

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

// simDuration is how long, in wall-clock time, a full simulated trip playback
// takes - independent of the trip's real DurationMin, so watching a demo trip
// doesn't take an hour. tickInterval is how often a position update is sent.
const (
	simDuration  = 60 * time.Second
	tickInterval = 500 * time.Millisecond
)

type positionUpdate struct {
	Lat              float64          `json:"lat"`
	Lon              float64          `json:"lon"`
	ProgressFraction float64          `json:"progress_fraction"`
	ETAMin           float64          `json:"eta_min"`
	Status           string           `json:"status"` // "in_progress" | "arrived"
	RestStop         *restStopPayload `json:"rest_stop,omitempty"`
}

type restStopPayload struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Name    string  `json:"name,omitempty"`
	Amenity string  `json:"amenity"`
}

var upgrader = websocket.Upgrader{
	// Demo/thesis project with a single trusted client (the Flutter app), not a
	// public multi-tenant service - a permissive CheckOrigin is an accepted scope cut.
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Gateway struct {
	Trips *store.TripStore
}

func New(trips *store.TripStore) *Gateway {
	return &Gateway{Trips: trips}
}

func (g *Gateway) HandleTripStream(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid trip id", http.StatusBadRequest)
		return
	}

	trip, err := g.Trips.Get(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load trip", http.StatusInternalServerError)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade failed for trip %d: %v", id, err)
		return
	}
	defer conn.Close()

	g.simulate(r.Context(), conn, trip)
}

func (g *Gateway) simulate(ctx context.Context, conn *websocket.Conn, trip store.Trip) {
	points := valhalla.DecodePolyline6(trip.Shape)
	if len(points) == 0 {
		_ = conn.WriteJSON(map[string]string{"error": "route has no geometry to simulate"})
		return
	}

	steps := int(simDuration / tickInterval)
	if steps < 1 {
		steps = 1
	}
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	restStopSent := false

	for i := 0; i <= steps; i++ {
		fraction := float64(i) / float64(steps)
		point := valhalla.PointAtFraction(points, fraction)

		update := positionUpdate{
			Lat:              point.Lat,
			Lon:              point.Lon,
			ProgressFraction: fraction,
			ETAMin:           trip.DurationMin * (1 - fraction),
			Status:           "in_progress",
		}
		if fraction >= 1 {
			update.Status = "arrived"
		}

		// Check once whether the trip.started worker has attached a rest-stop
		// suggestion yet, and surface it the first time it's there - avoids
		// re-sending it on every single tick once it exists.
		if !restStopSent {
			if current, err := g.Trips.Get(ctx, trip.ID); err == nil && current.RestStopLat != nil {
				update.RestStop = &restStopPayload{
					Lat:     *current.RestStopLat,
					Lon:     *current.RestStopLon,
					Amenity: derefOr(current.RestStopAmenity, ""),
					Name:    derefOr(current.RestStopName, ""),
				}
				restStopSent = true
			}
		}

		if err := conn.WriteJSON(update); err != nil {
			return // client disconnected
		}
		if fraction >= 1 {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func derefOr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}
