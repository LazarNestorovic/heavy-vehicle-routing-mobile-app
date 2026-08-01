// Package ws is the WebSocket gateway from SPECIFIKACIJA.md 3.7: relays a
// vehicle's real GPS position to every connected client for a trip (the
// driver's own screen and, potentially, their dispatcher's live map). A
// connection simply waits for the driver's phone to report a position
// (ReportPosition, called from POST /api/v1/trips/{id}/position) - there is
// no simulated fallback; an earlier version of this gateway synthesized a
// walk along the route's shape when no real ping had arrived yet, but that
// produced a visible "jump" whenever the real first GPS fix landed a moment
// later than the fallback kicked in (documentations/fixes/2026-07-26-*.md),
// and by this stage of the app a real GPS fix is already required to start a
// trip at all (see widgets/start_proximity_status.dart) - see
// documentations/fixes/2026-07-28-remove-simulated-fallback.md.
package ws

import (
	"context"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"

	"heavy-vehicle-routing/backend/internal/store"
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
// one trip. Created the moment EITHER side shows up first - a WS connection
// opening (which must subscribe before it can wait for anything, see
// HandleTripStream) or ReportPosition landing before anyone's watching yet.
type liveTrip struct {
	mu          sync.Mutex
	subscribers map[chan positionUpdate]struct{}
	last        *positionUpdate // most recent broadcast, replayed to new subscribers - see subscribe()
}

func newLiveTrip() *liveTrip {
	return &liveTrip{subscribers: make(map[chan positionUpdate]struct{})}
}

// subscribe registers ch and, if a position has already been reported,
// immediately replays it. Without this, a dispatcher who leaves and reopens
// the live map (or any client reconnecting) would see nothing until the
// vehicle's NEXT GPS fix - which can be minutes away, since fixes only arrive
// every distanceFilter meters of movement (LocationService.positionStream)
// and don't fire at all while the vehicle is stationary. See
// documentations/fixes/2026-08-01-dispatcher-live-map-loses-position-on-reopen.md.
func (lt *liveTrip) subscribe() chan positionUpdate {
	ch := make(chan positionUpdate, 4)
	lt.mu.Lock()
	lt.subscribers[ch] = struct{}{}
	last := lt.last
	lt.mu.Unlock()
	if last != nil {
		select {
		case ch <- *last:
		default: // unreachable for a fresh channel, kept for consistency with broadcast()
		}
	}
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
	lt.last = &update
	for ch := range lt.subscribers {
		select {
		case ch <- update:
		default: // a slow/stuck subscriber shouldn't block the reporter
		}
	}
}

type Gateway struct {
	Trips *store.TripStore

	liveMu sync.Mutex
	live   map[int64]*liveTrip
}

func New(trips *store.TripStore) *Gateway {
	return &Gateway{Trips: trips, live: make(map[int64]*liveTrip)}
}

// liveTripFor always returns tripID's shared liveTrip object, creating it if
// this is the first time anyone (a WS connection or ReportPosition) has
// touched this trip.
func (g *Gateway) liveTripFor(tripID int64) *liveTrip {
	g.liveMu.Lock()
	defer g.liveMu.Unlock()
	lt, ok := g.live[tripID]
	if !ok {
		lt = newLiveTrip()
		g.live[tripID] = lt
	}
	return lt
}

// ReportPosition is called from the driver's phone (POST /api/v1/trips/{id}/position)
// with a real GPS fix, and broadcasts it to every WS watcher of that trip
// (see HandleTripStream). Progress/ETA are approximated from the remaining
// haversine distance to the destination at the trip's originally-planned
// average pace - the same kind of honest simplification as
// valhalla.PointAtFraction's constant-speed assumption
// (documentations/features/2026-07-21-rest-stop-locations.md).
func (g *Gateway) ReportPosition(trip store.Trip, lat, lon float64) {
	lt := g.liveTripFor(trip.ID)

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
// auto-arrival signal, so this is driven by an explicit
// POST /api/v1/trips/{id}/complete from the driver instead.
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

// HandleTripStream subscribes the caller to trip id's live position feed and
// relays every update until the trip arrives or the client disconnects.
// Subscribing happens before anything else so no ReportPosition broadcast
// can land in a gap between "check if live data exists" and "start
// listening" - liveTrip.broadcast() is non-blocking and unbuffered, so a
// send with no subscriber yet would otherwise be silently dropped forever.
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

	g.relayLive(r.Context(), conn, g.liveTripFor(trip.ID))
}

// relayLive subscribes to lt and forwards every broadcast to conn until the
// trip arrives, the client disconnects, or ctx is done.
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
