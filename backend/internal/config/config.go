package config

import "os"

type Config struct {
	Port             string
	ValhallaURL      string
	DatabaseURL      string
	RabbitMQURL      string
	RestStopDataPath string
}

func Load() Config {
	return Config{
		Port:             getEnv("PORT", "8080"),
		ValhallaURL:      getEnv("VALHALLA_URL", "http://localhost:8002"),
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RestStopDataPath: getEnv("REST_STOP_DATA_PATH", "data/serbia-rest-stops.osm"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
