package twilio

import (
	appModule "diaxel/internal/app"
	"github.com/gin-gonic/gin"
)

func TwilioWebhookRoutes(router *gin.Engine, app *appModule.App) {
	twilioWebhookHandler := NewTwilioWebhookHandler(app.Cfg, app.LLM, app.Twilio, app.Db)
	productGroup := router.Group("twilio")
	{
		productGroup.POST("/webhook/:assistant_id", twilioWebhookHandler.HandleWebhook)
	}
}
