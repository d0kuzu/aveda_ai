package test

import (
	"diaxel/internal/modules/googlecalendar"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type TestHandler struct {
	gc *googlecalendar.Client
}

func NewTestHandler(gc *googlecalendar.Client) *TestHandler {
	return &TestHandler{gc: gc}
}

func (h *TestHandler) TestCalendar(c *gin.Context) {
	if h.gc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Google Calendar client is not initialized",
		})
		return
	}

	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	// Get freebusy status for the primary calendar to verify API client is working
	resp, err := h.gc.GetFreeBusy("primary", now, tomorrow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to verify Google Calendar connection",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  "Google Calendar client successfully connected and authorized",
		"freebusy": resp,
	})
}
