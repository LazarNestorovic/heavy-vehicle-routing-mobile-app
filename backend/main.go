package main

import (
	"context"
	"log"
	"net/http"

	"heavy-vehicle-routing/backend/internal/auth"
	"heavy-vehicle-routing/backend/internal/config"
	"heavy-vehicle-routing/backend/internal/db"
	"heavy-vehicle-routing/backend/internal/explain"
	"heavy-vehicle-routing/backend/internal/httpapi"
	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/reststop"
	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
	"heavy-vehicle-routing/backend/internal/worker"
	"heavy-vehicle-routing/backend/internal/ws"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	conn, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(ctx, conn); err != nil {
		log.Fatalf("migrate db: %v", err)
	}

	// Separate connections for the HTTP server's publisher and the worker's consumer -
	// an amqp091-go Channel isn't safe for concurrent use across goroutines.
	publisherQueue, err := queue.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("connect rabbitmq (publisher): %v", err)
	}
	defer publisherQueue.Close()

	consumerQueue, err := queue.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("connect rabbitmq (consumer): %v", err)
	}
	defer consumerQueue.Close()

	restStops, err := reststop.Load(cfg.RestStopDataPath)
	if err != nil {
		log.Fatalf("load rest stops from %s: %v", cfg.RestStopDataPath, err)
	}
	log.Printf("loaded %d rest stops from %s", len(restStops), cfg.RestStopDataPath)
	restStopFinder := reststop.NewFinder(restStops)

	vhClient := valhalla.New(cfg.ValhallaURL)
	vehicles := store.NewVehicleStore(conn)
	trips := store.NewTripStore(conn)
	drivers := store.NewDriverStore(conn)
	preferences := store.NewPreferencesStore(conn)
	favoriteStops := store.NewFavoriteStopStore(conn)
	explainer := explain.New(vhClient)
	wsGateway := ws.New(trips)
	authManager := auth.New(cfg.JWTSecret)
	server := httpapi.NewServer(vhClient, vehicles, trips, drivers, preferences, favoriteStops, restStopFinder, publisherQueue, explainer, wsGateway, authManager)

	tripWorker := &worker.TripWorker{
		Trips: trips, Queue: consumerQueue, RestStops: restStopFinder,
		Preferences: preferences, FavoriteStops: favoriteStops,
	}
	go func() {
		if err := tripWorker.Run(ctx); err != nil {
			log.Fatalf("trip worker: %v", err)
		}
	}()

	log.Printf("hvr-backend listening on :%s (valhalla=%s)", cfg.Port, cfg.ValhallaURL)
	if err := http.ListenAndServe(":"+cfg.Port, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
