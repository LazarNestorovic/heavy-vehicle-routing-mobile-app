// Package worker consumes trip.started events and computes a simplified
// rest-stop suggestion: after restThresholdMin of driving (a stand-in for a real
// AETR-style rule, see SPECIFIKACIJA.md 3.8), find the nearest fuel/parking/
// rest-area OSM node to where the vehicle would roughly be at that point,
// assuming constant speed along the route (a simplification - real speed varies
// by road segment).
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/reststop"
	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

// restThresholdMin is a stand-in for a real AETR-style driving-hours rule: 4.5h.
const restThresholdMin = 270

// routeSampleStride matches scoring.go's shapeSampleStride - check every Nth
// decoded shape point when building the route corridor for
// reststop.Finder.NearestOnRoute, for performance.
const routeSampleStride = 5

type TripWorker struct {
	Trips         *store.TripStore
	TripEvents    *store.TripEventStore
	Queue         *queue.Client
	RestStops     *reststop.Finder
	Preferences   *store.PreferencesStore
	FavoriteStops *store.FavoriteStopStore
	Vehicles      *store.VehicleStore
}

// Run blocks, consuming trip.started events until the delivery channel closes.
func (w *TripWorker) Run(ctx context.Context) error {
	deliveries, err := w.Queue.Consume("trip-worker.trip-started", queue.RoutingKeyTripStarted)
	if err != nil {
		return err
	}

	for d := range deliveries {
		w.handle(ctx, d)
	}
	return nil
}

func (w *TripWorker) handle(ctx context.Context, d amqp.Delivery) {
	var evt queue.TripStartedEvent
	if err := json.Unmarshal(d.Body, &evt); err != nil {
		log.Printf("worker: bad trip.started payload: %v", err)
		_ = d.Nack(false, false)
		return
	}

	trip, err := w.Trips.Get(ctx, evt.TripID)
	if err != nil {
		log.Printf("worker: load trip %d: %v", evt.TripID, err)
		_ = d.Nack(false, false)
		return
	}

	rest := w.computeRestStop(ctx, trip)

	if err := w.Trips.UpdateAfterProcessing(ctx, trip.ID, rest); err != nil {
		log.Printf("worker: update trip %d: %v", trip.ID, err)
		_ = d.Nack(false, true) // our own DB hiccup, not a bad message - requeue
		return
	}

	if rest.AfterMinutes != nil {
		desc := fmt.Sprintf("Preporučena pauza za odmor nakon %.0f min", *rest.AfterMinutes)
		if _, err := w.TripEvents.Create(ctx, trip.ID, "rest_stop_suggested", desc); err != nil {
			log.Printf("worker: log rest_stop_suggested for trip %d: %v", trip.ID, err)
		}
	}

	_ = d.Ack(false)
	log.Printf("worker: processed trip %d (rest_suggestion_min=%v)", trip.ID, rest.AfterMinutes)
}

func (w *TripWorker) computeRestStop(ctx context.Context, trip store.Trip) store.RestStopSuggestion {
	if trip.DurationMin <= restThresholdMin {
		return store.RestStopSuggestion{}
	}

	afterMin := float64(restThresholdMin)
	suggestion := store.RestStopSuggestion{AfterMinutes: &afterMin}

	points := valhalla.DecodePolyline6(trip.Shape)
	fraction := restThresholdMin / trip.DurationMin
	at := valhalla.PointAtFraction(points, fraction)

	routePoints := make([]reststop.Point, 0, len(points)/routeSampleStride+1)
	for i := 0; i < len(points); i += routeSampleStride {
		routePoints = append(routePoints, reststop.Point{Lat: points[i].Lat, Lon: points[i].Lon})
	}

	var brand string
	var favorites []reststop.Stop
	if prefs, err := w.Preferences.Get(ctx, trip.DriverID); err != nil {
		log.Printf("worker: load preferences for driver %d: %v (falling back to plain nearest)", trip.DriverID, err)
	} else if prefs.PreferredFuelBrand != nil {
		brand = *prefs.PreferredFuelBrand
	}
	if favs, err := w.FavoriteStops.List(ctx, trip.DriverID); err != nil {
		log.Printf("worker: load favorite stops for driver %d: %v (falling back to plain nearest)", trip.DriverID, err)
	} else {
		for _, f := range favs {
			favorites = append(favorites, reststop.Stop{ID: f.ID, Lat: f.Lat, Lon: f.Lon, Name: f.Name})
		}
	}

	var hazmat bool
	if vehicle, err := w.Vehicles.Get(ctx, trip.VehicleID); err != nil {
		log.Printf("worker: load vehicle %d for hazmat check: %v (assuming non-hazmat)", trip.VehicleID, err)
	} else {
		hazmat = vehicle.Hazmat
	}

	stop, _, found := w.RestStops.NearestOnRoute(at.Lat, at.Lon, brand, favorites, reststop.DefaultPreferredRadiusM, routePoints, reststop.DefaultRouteCorridorRadiusM, hazmat)
	if !found {
		return suggestion // threshold still meaningful even without a matched location
	}

	suggestion.Lat = &stop.Lat
	suggestion.Lon = &stop.Lon
	suggestion.Amenity = &stop.Amenity
	if stop.Name != "" {
		suggestion.Name = &stop.Name
	}
	return suggestion
}
