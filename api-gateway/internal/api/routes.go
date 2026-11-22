package api

import (
	"net/http"

	"api-gateway/internal/api/middleware"
	"api-gateway/internal/app"
	"github.com/gin-gonic/gin"
)

func NewRouter(application *app.App) *gin.Engine {
	router := gin.New()
	router.Use(gin.LoggerWithFormatter(middleware.Formatter), gin.Recovery(), headers)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.Any("/auth/*any", gin.WrapH(http.StripPrefix("/auth", application.Auth)))
	router.POST("/webhooks/telegram", gin.WrapH(application.AI))
	router.POST("/webhooks/telegram/callback/:assistant_id", gin.WrapH(application.AI))
	router.POST("/twilio/webhook/:assistant_id", gin.WrapH(application.AI))

	cabinet := router.Group("/", middleware.UserMiddleware([]byte(application.Cfg.AccessSecret)))
	cabinet.Any("/ai/*any", gin.WrapH(http.StripPrefix("/ai", application.AI)))
	analytics := gin.WrapH(http.StripPrefix("/api", application.AI))
	cabinet.GET("/api/analytics", analytics)
	cabinet.GET("/api/analytics/*any", analytics)
	return router
}

func headers(c *gin.Context) {
	c.Request.Header.Del("X-User-Id")
	c.Request.Header.Del("X-User-Role")
	c.Request.Header.Del("X-Assistant-Id")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}
