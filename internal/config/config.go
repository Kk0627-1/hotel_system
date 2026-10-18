package config

import (
	"fmt"
	"os"
	"strconv"

	"hotelsystem/internal/platform/database"
)

type Config struct {
	HTTPPort string
	Database database.Config
}

func Load() (Config, error) {
	port, err := envInt("DB_PORT", 5432)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPPort: envString("HTTP_PORT", "8080"),
		Database: database.Config{
			Host:     envString("DB_HOST", "localhost"),
			Port:     port,
			User:     envString("DB_USER", "hotel_Manager"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     envString("DB_NAME", "hotel_manager"),
			SSLMode:  envString("DB_SSLMODE", "disable"),
		},
	}, nil
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return n, nil
}
