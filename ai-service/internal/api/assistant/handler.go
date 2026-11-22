package assistant

import (
	"net/http"

	"diaxel/internal/api/access"
	"diaxel/internal/config"
	"diaxel/internal/modules/database"

	"github.com/gin-gonic/gin"
)

type AssistantHandler struct {
	cfg *config.Settings
	db  *database.Store
}

func NewAssistantHandler(cfg *config.Settings, db *database.Store) *AssistantHandler {
	return &AssistantHandler{cfg: cfg, db: db}
}

func (h *AssistantHandler) GetAssistants(c *gin.Context) {
	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-Id header is required"})
		return
	}

	assistants, err := h.db.GetAssistantsByUserID(c.Request.Context(), userId)
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

	assistant, ok := access.Assistant(c, h.db, id)
	if !ok {
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

	userID := access.UserID(c)
	if userID == "" {
		return
	}

	resp, err := h.db.UpdateAssistantForUser(c.Request.Context(), id, userID, req.Name, req.Configuration)
	if !access.Found(c, err) {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"answer": resp,
	})
}
