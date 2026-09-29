package api

import (
	"diaxel/internal/api/assistant"
	"diaxel/internal/api/chat"
	"diaxel/internal/api/campuslogin"
	"diaxel/internal/api/google"
	"diaxel/internal/api/twilio"
	"diaxel/internal/api/webhook"
	"diaxel/internal/api/ws"
	"diaxel/internal/api/analytics"
	"diaxel/internal/api/blockedcustomers"
	"diaxel/internal/api/test"
	appModule "diaxel/internal/app"
	"log"

	"github.com/gin-gonic/gin"
)

func RouterStart(app *appModule.App) {
	r := gin.Default()

	// CORS is handled at API Gateway and Nginx reverse proxy level

	webhook.WebhookRoutes(r, app)
	twilio.TwilioWebhookRoutes(r, app)
	ws.WSRoutes(r, app)
	chat.ChatRoutes(r, app)
	assistant.AssistantRoutes(r, app)
	campuslogin.CampusLoginRoutes(r, app)
	analytics.AnalyticsRoutes(r, app)
	test.TestRoutes(r, app)
	google.GoogleRoutes(r, app)
	blockedcustomers.BlockedCustomersRoutes(r, app)

	err := r.Run(":" + app.Cfg.HTTPPort)
	if err != nil {
		log.Fatal("Router start error", err)
	}
}

