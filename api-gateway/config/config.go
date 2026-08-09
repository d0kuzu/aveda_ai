package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GatewayPort string
	GRPCAddress string

	UserServiceURL string
	AuthServiceURL string
	AIServiceURL   string

	AccessSecret string
}

func LoadConfig() (*Config, error) {
	godotenv.Load(".env") // ignore error — env vars may come from docker-compose

	grpcAddress := os.Getenv("GRPC_ADDRESS")
	if grpcAddress == "" {
		grpcAddress = "localhost:50051"
	}

	return &Config{
		GatewayPort: os.Getenv("GATEWAY_PORT"),
		GRPCAddress: grpcAddress,

		UserServiceURL: os.Getenv("USER_SERVICE_URL"),
		AuthServiceURL: os.Getenv("AUTH_SERVICE_URL"),
		AIServiceURL:   os.Getenv("AI_SERVICE_URL"),

		AccessSecret: os.Getenv("ACCESS_SECRET"),
	}, nil
}
