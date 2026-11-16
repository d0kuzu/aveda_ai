package twilio

import (
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/llm"
	twilio2 "diaxel/internal/modules/twilio"
	"net/http"
	"strings"

	"log"

	"github.com/gin-gonic/gin"
)

func forwardWebhookToCampusLogin(formData string) {
	req, err := http.NewRequest("POST", "https://voip.campuslogin.com/TextMessage/TextIncoming.asmx/Collector", strings.NewReader(formData))
	if err != nil {
		log.Printf("[Twilio Webhook Forwarder] Error creating request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Twilio Webhook Forwarder] Error forwarding request: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("[Twilio Webhook Forwarder] Forwarded request, status: %d", resp.StatusCode)
}

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
	// Parse form early to forward the exact payload
	if err := c.Request.ParseForm(); err == nil {
		formData := c.Request.PostForm.Encode()
		go forwardWebhookToCampusLogin(formData)
	} else {
		log.Printf("[Twilio Webhook] Warning: Failed to parse form for forwarding: %v", err)
	}

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

	isBlocked, err := h.db.IsCustomerBlocked(from)
	if err == nil && isBlocked {
		log.Printf("[Twilio Webhook] Ignoring message from blocked customer %s", from)
		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
		return
	}

	log.Printf("[Twilio Webhook] Received message from %s: %s", from, body)

	chat, err := h.db.GetLatestChatByCustomer(assistantID, from)
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

	if (chat == nil || chat.Id == "") && !strings.Contains(body, "3000") {
		log.Printf("[Twilio Webhook] Ignoring message from %s: no active chat found. Must be triggered via CampusLogin first.", from)
		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
		return
	}

	//if chat.IsReviewed {
	//	_, err := h.db.UpdateChatIsReviewed(chat.Id, false)
	//	if err != nil {
	//		log.Printf("[Twilio Webhook] Warning: Failed to reset is_reviewed for chat %s: %v", chat.Id, err)
	//	}
	//}

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
