package app

import (
	"diaxel/internal/config"
	"diaxel/internal/modules/database"
	"diaxel/internal/modules/llm"
	"diaxel/internal/modules/telegram"
	"diaxel/internal/modules/twilio"
)

type App struct {
	LLM            *llm.Client
	Twilio         *twilio.Client
	Db             *database.Store
	Cfg            *config.Settings
	TgOrchestrator *telegram.Orchestrator
}

func NewApp(
	llmClient *llm.Client,
	twilioClient *twilio.Client,
	db *database.Store,
	cfg *config.Settings,
	tgOrch *telegram.Orchestrator,
) *App {
	return &App{
		LLM:            llmClient,
		Twilio:         twilioClient,
		Db:             db,
		Cfg:            cfg,
		TgOrchestrator: tgOrch,
	}
}
