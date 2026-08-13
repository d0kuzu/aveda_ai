package google

import (
	"net/http"

	appModule "diaxel/internal/app"

	"github.com/gin-gonic/gin"
)

func CalendarAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientSecret := c.GetHeader("X-Aveda-Calendar-Secret")
		if secret != "" && clientSecret != secret {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func GoogleRoutes(router *gin.Engine, app *appModule.App) {
	handler := NewGoogleHandler(app.GoogleCalendar, app.Db, app.CampusLogin)
	calendarHandler := NewCalendarHandler(app.GoogleCalendar, app.Db, app.Cfg)

	googleGroup := router.Group("google")
	{
		googleGroup.POST("/webhook", handler.HandleWebhook)
		
		calendarGroup := googleGroup.Group("calendar")
		calendarGroup.Use(CalendarAuthMiddleware(app.Cfg.AvedaCalendarSecret))
		{
			calendarGroup.GET("/slots", calendarHandler.GetSlots)
			calendarGroup.POST("/book", calendarHandler.BookSlot)
		}
	}
}

