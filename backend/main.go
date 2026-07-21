package main

import (
	"log"
	"net/http"

	"heavy-vehicle-routing/backend/internal/config"
	"heavy-vehicle-routing/backend/internal/httpapi"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

func main() {
	cfg := config.Load()

	vhClient := valhalla.New(cfg.ValhallaURL)
	server := httpapi.NewServer(vhClient)

	log.Printf("hvr-backend listening on :%s (valhalla=%s)", cfg.Port, cfg.ValhallaURL)
	if err := http.ListenAndServe(":"+cfg.Port, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
