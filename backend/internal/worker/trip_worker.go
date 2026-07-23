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
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/reststop"
	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

// restThresholdMin is a stand-in for a real AETR-style driving-hours rule: 4.5h.
const restThresholdMin = 270

type TripWorker struct {
	Trips     *store.TripStore
	Queue     *queue.Client
	RestStops *reststop.Finder
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

	rest := w.computeRestStop(trip)

	if err := w.Trips.UpdateAfterProcessing(ctx, trip.ID, rest); err != nil {
		log.Printf("worker: update trip %d: %v", trip.ID, err)
		_ = d.Nack(false, true) // our own DB hiccup, not a bad message - requeue
		return
	}

	body, err := json.Marshal(queue.TripETAUpdatedEvent{
		TripID:                trip.ID,
		DurationMin:           trip.DurationMin,
		NextRestSuggestionMin: rest.AfterMinutes,
		RestStopLat:           rest.Lat,
		RestStopLon:           rest.Lon,
		RestStopName:          rest.Name,
		RestStopAmenity:       rest.Amenity,
	})
	if err != nil {
		log.Printf("worker: encode trip.eta_updated for %d: %v", trip.ID, err)
	} else if err := w.Queue.Publish(ctx, queue.RoutingKeyTripETAUpdated, body); err != nil {
		log.Printf("worker: publish trip.eta_updated for %d: %v", trip.ID, err)
	}

	_ = d.Ack(false)
	log.Printf("worker: processed trip %d (rest_suggestion_min=%v)", trip.ID, rest.AfterMinutes)
}

func (w *TripWorker) computeRestStop(trip store.Trip) store.RestStopSuggestion {
	if trip.DurationMin <= restThresholdMin {
		return store.RestStopSuggestion{}
	}

	afterMin := float64(restThresholdMin)
	suggestion := store.RestStopSuggestion{AfterMinutes: &afterMin}

	points := valhalla.DecodePolyline6(trip.Shape)
	fraction := restThresholdMin / trip.DurationMin
	at := valhalla.PointAtFraction(points, fraction)

	stop, _, found := w.RestStops.Nearest(at.Lat, at.Lon)
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
