package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Settings struct {
	HTTPPort    string
	DatabaseURL string

	OpenaiApiKey string

	TelegramWebhookSecret string
	WebhookBaseURL        string
	BaseURL               string

	TokenPrefix string
	TokenLength int
}

func LoadConfig() (*Settings, error) {
	godotenv.Load(".env")

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	return &Settings{
		HTTPPort:    httpPort,
		DatabaseURL: databaseURL,

		OpenaiApiKey: os.Getenv("OPENAI_API_KEY"),

		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		WebhookBaseURL:        os.Getenv("WEBHOOK_BASE_URL"),
		BaseURL:               os.Getenv("BASE_URL"),

		TokenPrefix: os.Getenv("TOKEN_PREFIX"),
		TokenLength: getEnvAsInt("TOKEN_LENGTH", 32),
	}, nil
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		return defaultVal
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultVal
	}

	return value
}
