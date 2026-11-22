package handlers

import (
	"net/http"

	"auth-service/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	access, refresh, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	access, refresh, err := h.auth.Refresh(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if c.ShouldBindJSON(&req) != nil {
		c.Status(http.StatusNoContent)
		return
	}

	h.auth.Logout(req.RefreshToken)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Register(c *gin.Context) {
	type RegisterRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	access, refresh, err := h.auth.Register(
		req.Email,
		req.Password,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	type RegisterResponse struct { //TODO перенести в DTO все структуры энпоинтов
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	})
}
