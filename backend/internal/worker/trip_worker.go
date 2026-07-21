// Package worker consumes trip.started events and computes a simplified
// rest-stop suggestion. This is a placeholder rule (see SPECIFIKACIJA.md 3.8) -
// no real driving-hours regulation or actual rest-area lookup yet, just a
// duration threshold, enough to prove the async trip.started -> trip.eta_updated
// flow end to end.
package worker

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/store"
)

// restThresholdMin is a stand-in for a real AETR-style driving-hours rule: 4.5h.
const restThresholdMin = 270

type TripWorker struct {
	Trips *store.TripStore
	Queue *queue.Client
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

	var restSuggestion *float64
	if trip.DurationMin > restThresholdMin {
		v := float64(restThresholdMin)
		restSuggestion = &v
	}

	if err := w.Trips.UpdateAfterProcessing(ctx, trip.ID, restSuggestion); err != nil {
		log.Printf("worker: update trip %d: %v", trip.ID, err)
		_ = d.Nack(false, true) // our own DB hiccup, not a bad message - requeue
		return
	}

	body, err := json.Marshal(queue.TripETAUpdatedEvent{
		TripID:                trip.ID,
		DurationMin:           trip.DurationMin,
		NextRestSuggestionMin: restSuggestion,
	})
	if err != nil {
		log.Printf("worker: encode trip.eta_updated for %d: %v", trip.ID, err)
	} else if err := w.Queue.Publish(ctx, queue.RoutingKeyTripETAUpdated, body); err != nil {
		log.Printf("worker: publish trip.eta_updated for %d: %v", trip.ID, err)
	}

	_ = d.Ack(false)
	log.Printf("worker: processed trip %d (rest_suggestion_min=%v)", trip.ID, restSuggestion)
}
