package google

import (
	appModule "diaxel/internal/app"

	"github.com/gin-gonic/gin"
)

func GoogleRoutes(router *gin.Engine, app *appModule.App) {
	handler := NewGoogleHandler(app.GoogleCalendar, app.Db)
	googleGroup := router.Group("google")
	{
		googleGroup.POST("/webhook", handler.HandleWebhook)
	}
}
