package routes

import (
	"api-gateway/config"
	"api-gateway/grpc/db"
	"api-gateway/middleware/auth"
	"api-gateway/middleware/logger"
	"api-gateway/proxy"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, cfg *config.Config, db *db.Client) {
	r.Use(gin.LoggerWithFormatter(logger.Formatter))

	public := r.Group("/")
	{
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		public.Any("/auth/*any",
			proxy.NewReverseProxy(cfg.AuthServiceURL, "/auth"),
		)

		public.Any("/webhooks/telegram/callback/*any",
			proxy.NewReverseProxy(cfg.AIServiceURL, ""),
		)

		public.Any("/twilio/webhook/*any",
			proxy.NewReverseProxy(cfg.AIServiceURL, ""),
		)

		public.Any("/campuslogin/test",
			proxy.NewReverseProxy(cfg.AIServiceURL, ""),
		)
	}

	userPrivate := r.Group("/")
	userPrivate.Use(auth.UserMiddleware([]byte(cfg.AccessSecret)))
	{
		userPrivate.Any("/users/*any",
			proxy.NewReverseProxy(cfg.UserServiceURL, "/users"),
		)

		userPrivate.Any("/api/analytics/*any",
			proxy.NewReverseProxy(cfg.AIServiceURL, "/api/analytics"),
		)

		userPrivate.Any("/ai/*any",
			proxy.NewReverseProxy(cfg.AIServiceURL, "/ai"),
		)
	}

	servicePrivate := r.Group("/")
	servicePrivate.Use(auth.ServiceMiddleware(db))
	{
		servicePrivate.Any("/webhooks/telegram",
			proxy.NewServiceProxy(cfg.AIServiceURL, ""),
		)
	}
}
