package twilio

import (
	"log"
	"net/http"

	"diaxel/internal/modules/database"
	"diaxel/internal/modules/llm"
	twilio2 "diaxel/internal/modules/twilio"

	"github.com/gin-gonic/gin"
)

type TwilioWebhookHandler struct {
	LLM    *llm.Client
	twilio *twilio2.Client
	db     *database.Store
}

func NewTwilioWebhookHandler(llmClient *llm.Client, twilioClient *twilio2.Client, dbClient *database.Store) *TwilioWebhookHandler {
	return &TwilioWebhookHandler{LLM: llmClient, twilio: twilioClient, db: dbClient}
}

func (h *TwilioWebhookHandler) HandleWebhook(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id is required"})
		return
	}

	_, err := h.db.GetAssistant(c.Request.Context(), assistantID)
	if err != nil {
		log.Printf("Error getting assistant %s: %v", assistantID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant not found"})
		return
	}

	twilioConfig, err := h.db.GetTwilioConfig(c.Request.Context(), assistantID)
	if err != nil {
		log.Printf("Error getting twilio config for assistant %s: %v", assistantID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "twilio configuration not found for this assistant"})
		return
	}

	from := c.PostForm("From")
	body := c.PostForm("Body")

	isBlocked, err := h.db.IsCustomerBlocked(c.Request.Context(), from)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not check blocked contacts"})
		return
	}
	if isBlocked {
		log.Printf("[Twilio Webhook] Ignoring message from blocked customer %s", from)
		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
		return
	}

	log.Printf("[Twilio Webhook] Received message from %s: %s", from, body)

	chat, err := h.db.GetLatestChatByCustomer(c.Request.Context(), assistantID, from)
	if err != nil {
		log.Printf("[Twilio Webhook] Error checking chat existence for %s: %v", from, err)
		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
		return
	}

	if chat != nil && chat.IsEnd {
		log.Printf("[Twilio Webhook] Ignoring message from %s: chat is ended.", from)
		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
		return
	}

	answer, err := h.LLM.Conversation(c.Request.Context(), from, assistantID, body)
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
