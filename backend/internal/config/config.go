package config

import "os"

type Config struct {
	Port             string
	ValhallaURL      string
	DatabaseURL      string
	RabbitMQURL      string
	RestStopDataPath string
	JWTSecret        string
	// GoogleClientID is the Web OAuth client ID from documentations/guides/
	// google-maps-setup.md step 7 - Google ID tokens must have this as their
	// `aud` claim (see internal/auth/google.go). Empty disables Google sign-in
	// (handleGoogleAuth will fail token verification for every request).
	GoogleClientID string
	// SMTP* configure internal/mailer for the email verification flow (see
	// documentations/guides/google-maps-setup.md step 8). Empty SMTPHost
	// disables actually sending mail - see mailer.Client for the no-op
	// fallback used in that case (so local dev without SMTP creds doesn't
	// crash registration, it just skips sending).
	SMTPHost         string
	SMTPPort         string
	SMTPUsername     string
	SMTPPassword     string
	SMTPFrom         string
	PublicBackendURL string
	// NominatimBaseURL/NominatimUserAgent configure internal/geocode (address
	// search - see documentations/features/ entry). Nominatim's usage policy
	// requires a real identifying User-Agent; override NOMINATIM_USER_AGENT
	// with contact info before any real (non-thesis-demo) deployment.
	NominatimBaseURL   string
	NominatimUserAgent string
}

func Load() Config {
	return Config{
		Port:             getEnv("PORT", "8080"),
		ValhallaURL:      getEnv("VALHALLA_URL", "http://localhost:8002"),
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RestStopDataPath: getEnv("REST_STOP_DATA_PATH", "data/serbia-rest-stops.osm"),
		// Insecure fallback for local `go run` without docker-compose; docker-compose.yml
		// sets a real random JWT_SECRET for the actual running stack.
		JWTSecret:          getEnv("JWT_SECRET", "dev-only-insecure-secret-change-me"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		SMTPHost:           getEnv("SMTP_HOST", ""),
		SMTPPort:           getEnv("SMTP_PORT", "587"),
		SMTPUsername:       getEnv("SMTP_USERNAME", ""),
		SMTPPassword:       getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:           getEnv("SMTP_FROM", "no-reply@hvr.local"),
		PublicBackendURL:   getEnv("PUBLIC_BACKEND_URL", "http://localhost:8080"),
		NominatimBaseURL:   getEnv("NOMINATIM_BASE_URL", "https://nominatim.openstreetmap.org"),
		NominatimUserAgent: getEnv("NOMINATIM_USER_AGENT", "heavy-vehicle-routing-thesis/1.0"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
