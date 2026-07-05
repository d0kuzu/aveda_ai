package test

import (
	appModule "diaxel/internal/app"

	"github.com/gin-gonic/gin"
)

func TestRoutes(router *gin.Engine, app *appModule.App) {
	h := NewTestHandler(app.GoogleCalendar)

	group := router.Group("test")
	{
		group.GET("/calendar", h.TestCalendar)
		group.POST("/calendar/event", h.TestCreateEvent)
	}
}
