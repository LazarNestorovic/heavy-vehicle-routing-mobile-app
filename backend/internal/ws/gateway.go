// Package ws is the WebSocket gateway from SPECIFIKACIJA.md 3.7: pushes a
// vehicle's position along a trip's route to a connected client. Two modes:
//   - Simulated (default): no real GPS ping has arrived yet for this trip, so
//     position is simulated by walking the route's decoded shape at a fixed
//     wall-clock pace - the original substitute for live GPS in a thesis demo
//     (documentations/features/2026-07-21-websocket-gateway.md).
//   - Live: once the driver's phone reports a real position (ReportPosition,
//     called from POST /api/v1/trips/{id}/position), all WS watchers for that
//     trip switch to relaying real pings instead - see documentations/features/
//     for the live-GPS feature entry.
package ws

import (
	"context"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
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

// liveTrip fans a real GPS ping out to every WS connection currently watching
// one trip (the driver's own screen and, potentially, their dispatcher's live
// map - both connect to the same GET /ws/trips/{id}).
type liveTrip struct {
	mu          sync.Mutex
	subscribers map[chan positionUpdate]struct{}
}

func newLiveTrip() *liveTrip {
	return &liveTrip{subscribers: make(map[chan positionUpdate]struct{})}
}

func (lt *liveTrip) subscribe() chan positionUpdate {
	ch := make(chan positionUpdate, 4)
	lt.mu.Lock()
	lt.subscribers[ch] = struct{}{}
	lt.mu.Unlock()
	return ch
}

func (lt *liveTrip) unsubscribe(ch chan positionUpdate) {
	lt.mu.Lock()
	delete(lt.subscribers, ch)
	lt.mu.Unlock()
	close(ch)
}

func (lt *liveTrip) broadcast(update positionUpdate) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	for ch := range lt.subscribers {
		select {
		case ch <- update:
		default: // a slow/stuck subscriber shouldn't block the reporter
		}
	}
}

type Gateway struct {
	Trips      *store.TripStore
	TripEvents *store.TripEventStore

	liveMu sync.Mutex
	live   map[int64]*liveTrip
}

func New(trips *store.TripStore, tripEvents *store.TripEventStore) *Gateway {
	return &Gateway{Trips: trips, TripEvents: tripEvents, live: make(map[int64]*liveTrip)}
}

func (g *Gateway) liveTripFor(tripID int64, create bool) *liveTrip {
	g.liveMu.Lock()
	defer g.liveMu.Unlock()
	lt, ok := g.live[tripID]
	if !ok && create {
		lt = newLiveTrip()
		g.live[tripID] = lt
	}
	return lt
}

// ReportPosition is called from the driver's phone (POST /api/v1/trips/{id}/position)
// with a real GPS fix. The first call for a trip switches every WS watcher of
// that trip from the simulated walk to live relaying (see HandleTripStream).
// Progress/ETA are approximated from the remaining haversine distance to the
// destination at the trip's originally-planned average pace - the same kind
// of honest simplification as PointAtFraction's constant-speed assumption
// (documentations/features/2026-07-21-rest-stop-locations.md).
func (g *Gateway) ReportPosition(trip store.Trip, lat, lon float64) {
	lt := g.liveTripFor(trip.ID, true)

	remainingKm := haversineKm(lat, lon, trip.DestinationLat, trip.DestinationLon)
	progress := 0.0
	if trip.DistanceKm > 0 {
		progress = 1 - remainingKm/trip.DistanceKm
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}
	}
	etaMin := trip.DurationMin
	if avgSpeedKmPerMin := trip.DistanceKm / trip.DurationMin; avgSpeedKmPerMin > 0 {
		etaMin = remainingKm / avgSpeedKmPerMin
	}

	update := positionUpdate{Lat: lat, Lon: lon, ProgressFraction: progress, ETAMin: etaMin, Status: "in_progress"}
	if trip.RestStopLat != nil && trip.RestStopLon != nil {
		update.RestStop = &restStopPayload{
			Lat:     *trip.RestStopLat,
			Lon:     *trip.RestStopLon,
			Amenity: derefOr(trip.RestStopAmenity, ""),
			Name:    derefOr(trip.RestStopName, ""),
		}
	}
	lt.broadcast(update)
}

// CompleteTrip marks a live-tracked trip as arrived: sends a final message to
// every watcher and tears down its live state. Real GPS has no reliable
// auto-arrival signal the way the simulated walk's progress_fraction=1 does,
// so this is driven by an explicit POST /api/v1/trips/{id}/complete from the
// driver instead.
func (g *Gateway) CompleteTrip(tripID int64) {
	g.liveMu.Lock()
	lt, ok := g.live[tripID]
	delete(g.live, tripID)
	g.liveMu.Unlock()
	if !ok {
		return
	}
	lt.broadcast(positionUpdate{Status: "arrived", ProgressFraction: 1})
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

	if lt := g.liveTripFor(trip.ID, false); lt != nil {
		g.relayLive(r.Context(), conn, lt)
		return
	}
	g.simulate(r.Context(), conn, trip)
}

// relayLive just forwards whatever ReportPosition/CompleteTrip broadcast to
// this one connection, until the trip arrives or the client disconnects.
func (g *Gateway) relayLive(ctx context.Context, conn *websocket.Conn, lt *liveTrip) {
	ch := lt.subscribe()
	defer lt.unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(update); err != nil {
				return
			}
			if update.Status == "arrived" {
				return
			}
		}
	}
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
		// A real GPS ping may have arrived mid-simulation (driver's app caught
		// up to the trip after starting the WS connection) - hand off to live
		// relaying rather than keep simulating alongside real pings.
		if lt := g.liveTripFor(trip.ID, false); lt != nil {
			g.relayLive(ctx, conn, lt)
			return
		}

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
			if _, err := g.TripEvents.Create(ctx, trip.ID, "arrived", "Arrived at destination"); err != nil {
				log.Printf("ws: log arrived event for trip %d: %v", trip.ID, err)
			}
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

const earthRadiusKm = 6371.0

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}
