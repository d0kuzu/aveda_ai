package app

import (
	"auth-service/internal/config"
	"auth-service/internal/modules/auth"
)

type App struct {
	Cfg  *config.Config
	Auth *auth.Service
}

func New(cfg *config.Config, service *auth.Service) *App {
	return &App{Cfg: cfg, Auth: service}
}
