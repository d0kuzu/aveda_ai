package test

import (
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/campuslogin"
	"diaxel/internal/modules/googlecalendar"
	"diaxel/internal/modules/llm"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type TestHandler struct {
	gc  *googlecalendar.Client
	cl  *campuslogin.Client
	db  *db.Client
	llm *llm.Client
}

func NewTestHandler(gc *googlecalendar.Client, cl *campuslogin.Client, dbClient *db.Client, llmClient *llm.Client) *TestHandler {
	return &TestHandler{gc: gc, cl: cl, db: dbClient, llm: llmClient}
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
			"error":   "start_time and end_time query parameters are required",
			"example": "/test/campuslogin/appointment?start_time=2026-05-19T10:00:00&end_time=2026-05-19T11:00:00&contact_id=12345&program_id=1&description=Test",
		})
		return
	}

	contactID, _ := strconv.Atoi(c.DefaultQuery("contact_id", "0"))
	programID, _ := strconv.Atoi(c.DefaultQuery("program_id", "0"))
	description := c.DefaultQuery("description", "Test Appointment from AI Service")

	err := h.cl.SendAppointment(c.Request.Context(), "Testing", startTime, endTime, contactID, programID, description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to send appointment",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Appointment sent successfully",
		"request_data": gin.H{
			"start_time":  startTime,
			"end_time":    endTime,
			"contact_id":  contactID,
			"program_id":  programID,
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

	c.JSON(http.StatusOK, gin.H{
		"status":          "success",
		"message":         "Events listed successfully",
		"events_count":    len(events),
		"next_sync_token": nextSyncToken,
		"events":          events,
	})
}

func (h *TestHandler) RetryAppointment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appointment id is required"})
		return
	}

	app, err := h.db.GetAppointmentByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}

	startTime, _ := time.Parse(time.RFC3339, app.StartTime)
	endTime, _ := time.Parse(time.RFC3339, app.EndTime)

	// We extract phone from description + title just like TrySendCampusLogin
	text := app.Description + " " + app.Title

	importRegexp := regexp.MustCompile(`\+?[1]?[-\s\.]?\(?\d{3}\)?[-\s\.]?\d{3}[-\s\.]?\d{4}`)
	nonDigit := regexp.MustCompile(`\D`)

	phoneStr := importRegexp.FindString(text)
	if phoneStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone not found in appointment text", "text": text})
		return
	}

	digits := nonDigit.ReplaceAllString(phoneStr, "")
	if len(digits) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone length extracted", "digits": digits})
		return
	}
	phoneSuffix := digits[len(digits)-10:]

	campusRecord, err := h.db.GetCampusloginByPhone(phoneSuffix)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found in CampusLogin by phone", "phoneSuffix": phoneSuffix})
		return
	}

	loc, err := time.LoadLocation("America/Winnipeg")
	if err != nil {
		loc = time.UTC
	}

	startTimeFormatted := startTime.In(loc).Format("2006-01-02T15:04:05")
	endTimeFormatted := endTime.In(loc).Format("2006-01-02T15:04:05")

	err = h.cl.SendAppointment(c.Request.Context(), "Campus Tour for "+campusRecord.FirstName, startTimeFormatted, endTimeFormatted, int(campusRecord.ContactId), int(campusRecord.ProgramId), app.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send appointment to CampusLogin API", "details": err.Error(), "campusRecord": campusRecord})
		return
	}

	// If successful, update DB
	h.db.UpdateAppointmentCampusloginStatus(id, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Appointment sent to CampusLogin successfully",
	})
}

type TestConversationRequest struct {
	AssistantID string `json:"assistant_id" form:"assistant_id"`
	From        string `json:"from" form:"from"`
	Message     string `json:"message" form:"message"`
	Trigger     bool   `json:"trigger" form:"trigger"`
	Name        string `json:"name" form:"name"`
}

func (h *TestHandler) TestConversation(c *gin.Context) {
	if h.llm == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM client is not initialized"})
		return
	}

	var req TestConversationRequest
	_ = c.ShouldBind(&req)

	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		assistantID = req.AssistantID
	}
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id is required"})
		return
	}

	from := req.From
	if from == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from is required"})
		return
	}

	var digitsOnly string
	for _, r := range from {
		if r >= '0' && r <= '9' {
			digitsOnly += string(r)
		}
	}
	if len(digitsOnly) == 10 {
		from = "+1" + digitsOnly
	} else if len(digitsOnly) == 11 && strings.HasPrefix(digitsOnly, "1") {
		from = "+" + digitsOnly
	} else if len(digitsOnly) > 0 && !strings.HasPrefix(from, "+") {
		from = "+" + from
	}

	isTrigger := req.Trigger || c.Query("trigger") == "true"
	if isTrigger {
		if h.db != nil {
			existingChat, checkErr := h.db.GetLatestChatByCustomer(assistantID, from)
			if checkErr == nil && existingChat != nil && existingChat.Id != "" {
				_ = h.db.DeleteChatAndMessages(existingChat.Id)
			}
		}

		name := req.Name
		if name == "" {
			name = "Lead"
		}

		systemPrompt := fmt.Sprintf(
			"This is a new lead. Name: %s. Greet them by name and send the standard initial outreach message offering program details (do not ask about speaking with advisors).",
			name,
		)

		log.Printf("[Test Conversation] Triggering initial outreach for %s, prompt: %s", from, systemPrompt)

		answer, err := h.llm.Conversation(c, from, assistantID, "", llm.WithSystemMessage(systemPrompt))
		if err != nil {
			log.Printf("[Test Conversation] LLM conversation error for assistant %s: %v", assistantID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":        "ok",
			"is_trigger":    true,
			"assistant_id":  assistantID,
			"from":          from,
			"system_prompt": systemPrompt,
			"answer":        answer,
		})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	log.Printf("[Test Conversation] Processing user message from %s for assistant %s: %s", from, assistantID, req.Message)

	answer, err := h.llm.Conversation(c, from, assistantID, req.Message)
	if err != nil {
		log.Printf("[Test Conversation] LLM conversation error for assistant %s: %v", assistantID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"is_trigger":   false,
		"assistant_id": assistantID,
		"from":         from,
		"message":      req.Message,
		"answer":       answer,
	})
}
