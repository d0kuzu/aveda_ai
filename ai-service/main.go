package main

import (
	"context"
	"log"

	"diaxel/internal/api"
	appModule "diaxel/internal/app"
	"diaxel/internal/config"
	"diaxel/internal/modules/database"
	"diaxel/internal/modules/llm"
	"diaxel/internal/modules/telegram"
	"diaxel/internal/modules/twilio"
)

func main() {
	settings, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := database.Open(ctx, settings.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	llmClient := llm.InitClient(settings.OpenaiApiKey, store, "")
	twilioClient := twilio.InitClient()
	tgOrchestrator := telegram.NewOrchestrator(llmClient, 5, 1000)
	go tgOrchestrator.Start(ctx)

	app := appModule.NewApp(llmClient, twilioClient, store, settings, tgOrchestrator)
	if err := api.NewRouter(app).Run(":" + settings.HTTPPort); err != nil {
		log.Fatal(err)
	}
}
