package test

import (
	"diaxel/internal/modules/campuslogin"
	"diaxel/internal/modules/googlecalendar"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type TestHandler struct {
	gc *googlecalendar.Client
	cl *campuslogin.Client
}

func NewTestHandler(gc *googlecalendar.Client, cl *campuslogin.Client) *TestHandler {
	return &TestHandler{gc: gc, cl: cl}
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

	event, err := h.gc.CreateSimpleEvent(req.Title, req.Start, req.End, "")
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

func (h *TestHandler) TestSendAppointment(c *gin.Context) {
	if h.cl == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CampusLogin client not initialized"})
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	if startTime == "" || endTime == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "start_time and end_time query parameters are required",
			"example": "/test/campuslogin/appointment?start_time=2026-05-19T10:00:00&end_time=2026-05-19T11:00:00&contact_id=12345&program_id=1&description=Test",
		})
		return
	}

	contactID, _ := strconv.Atoi(c.DefaultQuery("contact_id", "0"))
	programID, _ := strconv.Atoi(c.DefaultQuery("program_id", "0"))
	description := c.DefaultQuery("description", "Test Appointment from AI Service")

	err := h.cl.SendAppointment(c.Request.Context(), startTime, endTime, contactID, programID, description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"message": "Failed to send appointment",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"message": "Appointment sent successfully",
		"request_data": gin.H{
			"start_time": startTime,
			"end_time": endTime,
			"contact_id": contactID,
			"program_id": programID,
			"description": description,
		},
	})
}

func (h *TestHandler) TestListEvents(c *gin.Context) {
	if h.gc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Google Calendar client is not initialized",
		})
		return
	}

	calendarID := c.DefaultQuery("calendar_id", "primary")
	syncToken := c.Query("sync_token")

	events, nextSyncToken, err := h.gc.ListEvents(calendarID, syncToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to list events",
			"details": err.Error(),
		})
		return
	}

	// Ограничиваем количество возвращаемых событий, чтобы избежать огромных JSON и broken pipe
	var responseEvents []*calendar.Event
	if len(events) > 5 {
		responseEvents = events[:5]
	} else {
		responseEvents = events
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          "success",
		"message":         "Events listed successfully",
		"events_count":    len(events),
		"next_sync_token": nextSyncToken,
		"events_sample":   responseEvents,
	})
}
