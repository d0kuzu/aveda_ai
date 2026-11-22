package config

import (
	"errors"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort    string
	DatabaseURL string

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	AccessSecret  string
	RefreshSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env")
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	cfg := &Config{
		HTTPPort:    port,
		DatabaseURL: os.Getenv("DATABASE_URL"),

		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,

		AccessSecret:  os.Getenv("ACCESS_SECRET"),
		RefreshSecret: os.Getenv("REFRESH_SECRET"),
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.AccessSecret == "" || cfg.RefreshSecret == "" {
		return nil, errors.New("ACCESS_SECRET and REFRESH_SECRET are required")
	}
	return cfg, nil
}
