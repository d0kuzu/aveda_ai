package webhook

import (
	appModule "diaxel/internal/app"

	"github.com/gin-gonic/gin"
)

func WebhookRoutes(router *gin.Engine, app *appModule.App) {
	aiHandler := NewAIHandler(app.Cfg, app.LLM, app.Db, app.TgOrchestrator)

	webhookGroup := router.Group("webhooks")
	{
		webhookGroup.POST("/telegram/register", aiHandler.RegisterTelegramBot)
		webhookGroup.POST("/api/register", aiHandler.RegisterAPIBot)
		webhookGroup.POST("/telegram", aiHandler.SendMessage)

		webhookGroup.POST("/twilio/register", aiHandler.RegisterTwilioBot)

		webhookGroup.POST("/telegram/callback/:assistant_id", aiHandler.HandleTelegramWebhook)
	}
}
