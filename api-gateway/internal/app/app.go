package app

import (
	"net/http"

	"api-gateway/internal/config"
)

type App struct {
	Cfg  *config.Settings
	Auth http.Handler
	AI   http.Handler
}
