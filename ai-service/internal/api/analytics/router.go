package analytics

import (
	"diaxel/internal/app"

	"github.com/gin-gonic/gin"
)

func AnalyticsRoutes(r *gin.Engine, application *app.App) {
	api := r.Group("/analytics")
	{
		api.GET("", GetAnalytics(application))
		api.GET("/chats", GetAnalyticsChatsByCategory(application, ""))
		api.GET("/chats/started", GetStartedChats(application))
		api.GET("/chats/completed", GetCompletedChats(application))
		api.GET("/chats/booked", GetBookedChats(application))
	}
}
