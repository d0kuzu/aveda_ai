package assistant

import (
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AssistantHandler struct {
	cfg *config.Settings
	db  *db.Client
}

func NewAssistantHandler(cfg *config.Settings, db *db.Client) *AssistantHandler {
	return &AssistantHandler{cfg: cfg, db: db}
}

func (h *AssistantHandler) GetAssistants(c *gin.Context) {
	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-Id header is required"})
		return
	}

	assistants, err := h.db.GetAssistantsByUserID(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch assistants", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"answer": assistants,
	})
}

func (h *AssistantHandler) GetAssistant(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	assistant, err := h.db.GetAssistant(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch assistant", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"answer": assistant,
	})
}

func (h *AssistantHandler) UpdateAssistant(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	var req struct {
		Name          string `json:"name"`
		Configuration string `json:"configuration"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	existing, err := h.db.GetAssistant(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch existing assistant", "details": err.Error()})
		return
	}

	resp, err := h.db.UpdateAssistant(
		id,
		req.Name,
		req.Configuration,
		existing.ApiToken,
		existing.TelegramBotToken,
		existing.Type,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update assistant", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"answer": resp,
	})
}
