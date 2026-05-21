package campuslogin

import (
	"diaxel/internal/config"
	"diaxel/internal/constants"
	"diaxel/internal/grpc/db"
	campusloginModule "diaxel/internal/modules/campuslogin"
	"diaxel/internal/modules/llm"
	twilio "diaxel/internal/modules/twilio"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type CampusLoginHandler struct {
	cfg    *config.Settings
	db     *db.Client
	twilio *twilio.Client
	LLM    *llm.Client
}

func NewCampusLoginHandler(cfg *config.Settings, db *db.Client, twilioClient *twilio.Client, llmClient *llm.Client) *CampusLoginHandler {
	return &CampusLoginHandler{
		cfg:    cfg,
		db:     db,
		twilio: twilioClient,
		LLM:    llmClient,
	}
}

type CampusWebhookRequest struct {
	ContactID      string `form:"ContactID" json:"ContactID"`
	CampusID       string `form:"CampusID" json:"CampusID"`
	FirstName      string `form:"FirstName" json:"FirstName"`
	LastName       string `form:"Lastname" json:"Lastname"`
	AlternatePhone string `form:"alternatephone" json:"alternatephone"`
	Email          string `form:"Email" json:"Email"`
	StudentNumber  string `form:"StudentNumber" json:"StudentNumber"`
	ID             string `form:"ID" json:"ID"`
	ProgramID      string `form:"ProgramID" json:"ProgramID"`
}

func (h *CampusLoginHandler) HandleTriggerTwilio(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id parameter is required"})
		return
	}

	if assistantID == "test" {
		client := campusloginModule.NewClient(h.cfg.CampusLoginAPI)
		err := client.SendAppointment(c.Request.Context(), "2026-05-25T11:30:00", "2026-05-25T12:30:00", 5972449, 1, "Test appointment from API endpoint")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Test failed: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Test appointment successfully sent to CampusLogin"})
		return
	}

	var req CampusWebhookRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("[CampusLogin Trigger] Binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data: " + err.Error()})
		return
	}

	toPhone := req.AlternatePhone
	if toPhone == "" {
		log.Printf("[CampusLogin Trigger] Alternate phone number is missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "alternatephone is required"})
		return
	}

	contactIDInt := 0
	if req.ContactID != "" {
		parsed, err := strconv.Atoi(req.ContactID)
		if err == nil {
			contactIDInt = parsed
		} else {
			log.Printf("[CampusLogin Trigger] Failed to parse ContactID '%s': %v", req.ContactID, err)
		}
	}

	var digitsOnly string
	for _, r := range toPhone {
		if r >= '0' && r <= '9' {
			digitsOnly += string(r)
		}
	}

	if len(digitsOnly) == 10 {
		toPhone = "+1" + digitsOnly
	} else if len(digitsOnly) == 11 && strings.HasPrefix(digitsOnly, "1") {
		toPhone = "+" + digitsOnly
	} else {
		// Fallback
		if !strings.HasPrefix(toPhone, "+") {
			toPhone = "+" + toPhone
		}
	}

	programIDInt := 0
	if req.ProgramID != "" {
		parsed, err := strconv.Atoi(req.ProgramID)
		if err == nil {
			programIDInt = parsed
		} else {
			log.Printf("[CampusLogin Trigger] Failed to parse ProgramID '%s': %v", req.ProgramID, err)
		}
	}

	if contactIDInt > 0 {
		err := h.db.UpsertCampuslogin(toPhone, contactIDInt, programIDInt)
		if err != nil {
			log.Printf("[CampusLogin Trigger] Failed to upsert Campuslogin for %s: %v", toPhone, err)
		}
	}

	if toPhone != "+12048176146" && toPhone != "+12045909711" && toPhone != "+12045589015" {
		log.Printf("[CampusLogin Trigger] Ignoring message from unknown number %s", toPhone)
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "message": "phone number not in allowed list"})
		return
	}

	_, err := h.db.GetAssistant(assistantID)
	if err != nil {
		log.Printf("Error getting assistant %s: %v", assistantID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant not found"})
		return
	}

	twilioConfig, err := h.db.GetTwilioConfig(assistantID)
	if err != nil {
		log.Printf("Error getting twilio config for assistant %s: %v", assistantID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "twilio configuration not found for this assistant"})
		return
	}

	programName := req.ProgramID
	if name, ok := constants.ProgramIDToName[req.ProgramID]; ok {
		programName = name
	}

	systemPrompt := fmt.Sprintf(
		"This is a new lead. Name: %s, program: %s. Greet them by name and mention the program they chose.",
		req.FirstName,
		programName,
	)

	answer, err := h.LLM.Conversation(c, toPhone, assistantID, "", llm.WithSystemMessage(systemPrompt))
	if err != nil {
		log.Printf("LLM Conversation error for assistant %s: %v", assistantID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = h.twilio.SendMessage(c,
		twilioConfig.AccountSid,
		twilioConfig.AuthToken,
		twilioConfig.TwilioNumber,
		toPhone,
		answer,
	)
	if err != nil {
		log.Printf("Error sending Twilio message to %s: %v", toPhone, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to send message: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Triggered successfully",
		"to":      toPhone,
		"answer":  answer,
	})
}
