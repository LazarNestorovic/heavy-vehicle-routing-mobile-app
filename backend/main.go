package main

import (
	"context"
	"log"
	"net/http"

	"heavy-vehicle-routing/backend/internal/config"
	"heavy-vehicle-routing/backend/internal/db"
	"heavy-vehicle-routing/backend/internal/httpapi"
	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
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

	vhClient := valhalla.New(cfg.ValhallaURL)
	vehicles := store.NewVehicleStore(conn)
	trips := store.NewTripStore(conn)
	server := httpapi.NewServer(vhClient, vehicles, trips)

	log.Printf("hvr-backend listening on :%s (valhalla=%s)", cfg.Port, cfg.ValhallaURL)
	if err := http.ListenAndServe(":"+cfg.Port, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
