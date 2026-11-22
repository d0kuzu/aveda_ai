package access

import (
	"errors"
	"net/http"

	"diaxel/internal/modules/database"

	"github.com/gin-gonic/gin"
)

// UserID reads the identity set by the gateway after verifying the user's JWT.
func UserID(c *gin.Context) string {
	id := c.GetHeader("X-User-Id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-User-Id header is required"})
	}
	return id
}

func Found(c *gin.Context, err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, database.ErrNotFound) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "resource not found"})
	} else {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not access resource"})
	}
	return false
}

func Assistant(c *gin.Context, db *database.Store, id string) (*database.Assistant, bool) {
	userID := UserID(c)
	if userID == "" {
		return nil, false
	}
	assistant, err := db.GetAssistantForUser(c.Request.Context(), id, userID)
	return assistant, Found(c, err)
}

func Chat(c *gin.Context, db *database.Store, id string) bool {
	userID := UserID(c)
	if userID == "" {
		return false
	}
	_, err := db.GetChatForUser(c.Request.Context(), id, userID)
	return Found(c, err)
}
