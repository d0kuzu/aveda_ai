package api

import (
	"auth-service/internal/api/handlers"
	"auth-service/internal/app"

	"github.com/gin-gonic/gin"
)

func NewRouter(a *app.App) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	h := handlers.NewAuthHandler(a.Auth)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/logout", h.Logout)
	r.POST("/register", h.Register)
	return r
}
