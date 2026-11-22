package main

import (
	"log"

	"api-gateway/internal/api"
	"api-gateway/internal/app"
	"api-gateway/internal/config"
	"api-gateway/internal/modules/proxy"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	authProxy, err := proxy.New(cfg.AuthServiceURL)
	if err != nil {
		log.Fatalf("AUTH_SERVICE_URL: %v", err)
	}
	aiProxy, err := proxy.New(cfg.AIServiceURL)
	if err != nil {
		log.Fatalf("AI_SERVICE_URL: %v", err)
	}

	application := &app.App{Cfg: cfg, Auth: authProxy, AI: aiProxy}
	log.Fatal(api.NewRouter(application).Run(":" + cfg.HTTPPort))
}
