package campuslogin

import (
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/llm"
	twilio "diaxel/internal/modules/twilio"
	"fmt"
	"log"
	"net/http"
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

	if !strings.HasPrefix(toPhone, "+") {
		toPhone = "+" + toPhone
	}

	if toPhone != "+16692430929" && toPhone != "+77002744195" {
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

	systemPrompt := fmt.Sprintf(
		"This is a new lead. Name: %s %s, program: %s. Greet them by name and mention the program they chose.",
		req.FirstName,
		req.LastName,
		req.ProgramID,
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
