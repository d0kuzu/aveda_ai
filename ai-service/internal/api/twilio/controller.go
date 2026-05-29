package twilio

import (
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/llm"
	twilio2 "diaxel/internal/modules/twilio"
	"net/http"

	"log"

	"github.com/gin-gonic/gin"
)

type TwilioWebhookHandler struct {
	cfg    *config.Settings
	LLM    *llm.Client
	twilio *twilio2.Client
	db     *db.Client
}

func NewTwilioWebhookHandler(cfg *config.Settings, llmClient *llm.Client, twilioClient *twilio2.Client, dbClient *db.Client) *TwilioWebhookHandler {
	return &TwilioWebhookHandler{cfg: cfg, LLM: llmClient, twilio: twilioClient, db: dbClient}
}

func (h *TwilioWebhookHandler) HandleWebhook(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id is required"})
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

	from := c.PostForm("From")
	body := c.PostForm("Body")

	log.Printf("[Twilio Webhook] Received message from %s: %s", from, body)

	chat, err := h.db.GetLatestChatByCustomer(assistantID, from)
	if err != nil {
		log.Printf("[Twilio Webhook] Error checking chat existence for %s: %v", from, err)
		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
		return
	}

	if chat == nil || chat.Id == "" {
		log.Printf("[Twilio Webhook] Ignoring message from %s: no active chat found. Must be triggered via CampusLogin first.", from)
		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
		return
	}

	answer, err := h.LLM.Conversation(c, from, assistantID, body)
	if err != nil {
		log.Printf("LLM Conversation error for assistant %s: %v", assistantID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[Twilio Webhook] Sending reply from %s to %s via REST API", twilioConfig.TwilioNumber, from)

	err = h.twilio.SendMessage(c,
		twilioConfig.AccountSid,
		twilioConfig.AuthToken,
		twilioConfig.TwilioNumber,
		from,
		answer,
	)
	if err != nil {
		log.Printf("Twilio SendMessage error for assistant %s: %v", assistantID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/xml")
	c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
}
