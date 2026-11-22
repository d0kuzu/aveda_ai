package main

import (
	"context"
	"log"

	"diaxel/internal/api"
	appModule "diaxel/internal/app"
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/llm"
	"diaxel/internal/modules/telegram"
	"diaxel/internal/modules/twilio"
)

func main() {
	settings, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	grpcClient, err := db.New(settings.GRPCAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer grpcClient.Close()

	llmClient := llm.InitClient(settings.OpenaiApiKey, grpcClient, "")
	twilioClient := twilio.InitClient()
	tgOrchestrator := telegram.NewOrchestrator(llmClient, 5, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go tgOrchestrator.Start(ctx)

	app := appModule.NewApp(llmClient, twilioClient, grpcClient, settings, tgOrchestrator)
	api.RouterStart(app)
}
