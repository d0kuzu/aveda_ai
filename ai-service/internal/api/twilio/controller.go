package twilio

import (
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/llm"
	twilio2 "diaxel/internal/modules/twilio"
	"net/http"

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
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant not found"})
		return
	}

	twilioConfig, err := h.db.GetTwilioConfig(assistantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "twilio configuration not found for this assistant"})
		return
	}

	from := c.PostForm("From")
	body := c.PostForm("Body")

	answer, err := h.LLM.Conversation(c, from, assistantID, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = h.twilio.SendMessage(c,
		twilioConfig.AccountSid,
		twilioConfig.AuthToken,
		twilioConfig.TwilioNumber,
		from,
		answer,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.XML(http.StatusOK, gin.H{
		"Response": gin.H{
			"Message": "OK",
		},
	})
}
