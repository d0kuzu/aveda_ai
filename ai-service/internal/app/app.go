package app

import (
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/campuslogin"
	"diaxel/internal/modules/googlecalendar"
	"diaxel/internal/modules/llm"
	"diaxel/internal/modules/telegram"
	"diaxel/internal/modules/twilio"
)

type LLM interface {
}

type Twilio interface {
}

type App struct {
	LLM            *llm.Client
	Twilio         *twilio.Client
	Db             *db.Client
	Cfg            *config.Settings
	TgOrchestrator *telegram.Orchestrator
	GoogleCalendar *googlecalendar.Client
	CampusLogin    *campuslogin.Client
}

func NewApp(
	llmClient *llm.Client,
	twilioClient *twilio.Client,
	db *db.Client,
	cfg *config.Settings,
	tgOrch *telegram.Orchestrator,
	gcClient *googlecalendar.Client,
	campusLoginClient *campuslogin.Client,
) *App {
	return &App{
		LLM:            llmClient,
		Twilio:         twilioClient,
		Db:             db,
		Cfg:            cfg,
		TgOrchestrator: tgOrch,
		GoogleCalendar: gcClient,
		CampusLogin:    campusLoginClient,
	}
}

