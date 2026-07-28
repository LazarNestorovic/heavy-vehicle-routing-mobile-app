package main

import (
	"context"
	"log"
	"net/http"

	"heavy-vehicle-routing/backend/internal/auth"
	"heavy-vehicle-routing/backend/internal/config"
	"heavy-vehicle-routing/backend/internal/db"
	"heavy-vehicle-routing/backend/internal/explain"
	"heavy-vehicle-routing/backend/internal/geocode"
	"heavy-vehicle-routing/backend/internal/httpapi"
	"heavy-vehicle-routing/backend/internal/mailer"
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
	tripEvents := store.NewTripEventStore(conn)
	drivers := store.NewDriverStore(conn)
	dispatcherRequests := store.NewDispatcherRequestStore(conn)
	preferences := store.NewPreferencesStore(conn)
	favoriteStops := store.NewFavoriteStopStore(conn)
	chats := store.NewChatMessageStore(conn)
	emailVerifications := store.NewEmailVerificationTokenStore(conn)
	passwordResets := store.NewPasswordResetTokenStore(conn)
	geocoder := geocode.New(cfg.NominatimBaseURL, cfg.NominatimUserAgent)
	explainer := explain.New(vhClient)
	wsGateway := ws.New(trips)
	chatWS := ws.NewChat(publisherQueue)
	authManager := auth.New(cfg.JWTSecret)
	mailerClient := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	if !mailerClient.Enabled() {
		log.Printf("email verification disabled (SMTP_HOST not set) - see documentations/guides/google-maps-setup.md step 8")
	}

	// Google sign-in is optional - if it's not provisioned yet (see
	// documentations/guides/google-maps-setup.md) or the JWKS fetch fails, the
	// rest of the app still starts normally; handleGoogleAuth just refuses
	// requests until it's configured.
	var googleAuth *auth.GoogleVerifier
	if cfg.GoogleClientID == "" {
		log.Printf("google sign-in disabled (GOOGLE_CLIENT_ID not set)")
	} else if v, err := auth.NewGoogleVerifier(ctx, cfg.GoogleClientID); err != nil {
		log.Printf("google sign-in disabled (failed to fetch Google JWKS): %v", err)
	} else {
		googleAuth = v
	}

	server := httpapi.NewServer(vhClient, vehicles, trips, tripEvents, drivers, dispatcherRequests, preferences, favoriteStops, chats, emailVerifications, passwordResets, restStopFinder, publisherQueue, explainer, wsGateway, chatWS, authManager, googleAuth, mailerClient, geocoder, cfg.PublicBackendURL)

	tripWorker := &worker.TripWorker{
		Trips: trips, TripEvents: tripEvents, Queue: consumerQueue, RestStops: restStopFinder,
		Preferences: preferences, FavoriteStops: favoriteStops, Vehicles: vehicles,
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
