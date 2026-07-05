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

type CreateEventRequest struct {
	Title string    `json:"title" binding:"required"`
	Start time.Time `json:"start" binding:"required"`
	End   time.Time `json:"end" binding:"required"`
}

func (h *TestHandler) TestCreateEvent(c *gin.Context) {
	if h.gc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Google Calendar client is not initialized",
		})
		return
	}

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Fallback for simple testing if no valid JSON is provided
		req.Title = "Test Event from AI Service"
		req.Start = time.Now().Add(1 * time.Hour)
		req.End = req.Start.Add(1 * time.Hour)
	}

	event, err := h.gc.CreateSimpleEvent(req.Title, req.Start, req.End)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create event",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Event created successfully",
		"event":   event,
	})
}
