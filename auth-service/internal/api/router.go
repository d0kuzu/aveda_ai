package api

import (
	"auth-service/internal/api/handlers"
	"auth-service/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, auth *service.AuthService) {
	h := handlers.NewAuthHandler(auth)

	authGroup := r.Group("/")
	{
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.Refresh)
		authGroup.POST("/logout", h.Logout)
		authGroup.POST("/register", h.Register)
	}
}
