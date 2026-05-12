package webhook

import (
	"bytes"
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/llm"
	"diaxel/internal/modules/token"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"diaxel/internal/modules/telegram"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	cfg          *config.Settings
	LLM          *llm.Client
	db           *db.Client
	orchestrator *telegram.Orchestrator
}

func NewAIHandler(cfg *config.Settings, llmClient *llm.Client, db *db.Client, tgOrch *telegram.Orchestrator) *AIHandler {
	return &AIHandler{cfg: cfg, LLM: llmClient, db: db, orchestrator: tgOrch}
}

type RegisterBotRequest struct {
	Name     string `json:"name"`
	BotToken string `json:"bot_token,omitempty"`
}

type RegisterTwilioBotRequest struct {
	Name         string `json:"name"`
	TwilioNumber string `json:"twilio_number"`
	AccountSID   string `json:"account_sid"`
	AuthToken    string `json:"auth_token"`
}

func (h *AIHandler) RegisterTelegramBot(c *gin.Context) {
	var req RegisterBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if req.BotToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bot_token is required"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized request from gateway"})
		return
	}

	assistant, err := h.db.CreateAssistant(req.Name, "", userId, req.BotToken, "telegram")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assistant in database"})
		return
	}

	webhookURL := fmt.Sprintf("%s/webhooks/telegram/callback/%s", h.cfg.WebhookBaseURL, assistant.Id)
	if err := h.setTelegramWebhook(req.BotToken, webhookURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to register webhook in Telegram: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "Telegram bot registered",
		"assistant_id": assistant.Id,
	})
}

func (h *AIHandler) setTelegramWebhook(botToken, webhookURL string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", botToken)

	body, err := json.Marshal(map[string]string{
		"url":          webhookURL,
		"secret_token": h.cfg.TelegramWebhookSecret,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal webhook request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to call Telegram API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API returned status %d", resp.StatusCode)
	}

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode Telegram response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("Telegram setWebhook failed: %s", result.Description)
	}

	return nil
}

func (h *AIHandler) RegisterAPIBot(c *gin.Context) {
	var req RegisterBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized request from gateway"})
		return
	}

	secureToken, err := token.GenerateSecureToken(h.cfg.TokenPrefix, h.cfg.TokenLength)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	assistant, err := h.db.CreateAssistant(req.Name, secureToken, userId, "", "api")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assistant in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Assistant registered", "assistant_id": assistant.Id, "token": secureToken})
}

func (h *AIHandler) RegisterTwilioBot(c *gin.Context) {
	var req RegisterTwilioBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if req.Name == "" || req.TwilioNumber == "" || req.AccountSID == "" || req.AuthToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "all fields (name, twilio_number, account_sid, auth_token) are required"})
		return
	}

	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized request from gateway"})
		return
	}

	assistant, err := h.db.CreateAssistant(req.Name, "", userId, "", "twilio")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assistant in database"})
		return
	}

	_, err = h.db.SaveTwilioConfig(assistant.Id, userId, req.TwilioNumber, req.AccountSID, req.AuthToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Twilio configuration"})
		return
	}

	webhookURL := fmt.Sprintf("%s/twilio/webhook/%s", h.cfg.BaseURL, assistant.Id)

	c.JSON(http.StatusOK, gin.H{
		"status":       "Twilio assistant registered",
		"assistant_id": assistant.Id,
		"webhook_url":  webhookURL,
	})
}

type TelegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
	} `json:"message"`
}

func (h *AIHandler) HandleTelegramWebhook(c *gin.Context) {
	assistantId := c.Param("assistant_id")

	secretToken := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	if h.cfg.TelegramWebhookSecret != "" && secretToken != h.cfg.TelegramWebhookSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid secret token"})
		return
	}

	var update TelegramUpdate

	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ignored - bad request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})

	if update.Message.Text == "" {
		return
	}

	userIdString := strconv.FormatInt(update.Message.From.ID, 10)
	if userIdString == "0" {
		return
	}

	assistant, err := h.db.GetAssistant(assistantId)
	if err != nil {
		fmt.Printf("HandleTelegramWebhook: Failed to get assistant: %v\n", err)
		return
	}

	if assistant.TelegramBotToken == "" {
		fmt.Printf("HandleTelegramWebhook: Assistant %s does not have a telegram bot token configured\n", assistantId)
		return
	}

	task := telegram.TelegramTask{
		BotToken:    assistant.TelegramBotToken,
		ChatID:      update.Message.Chat.ID,
		UserID:      userIdString,
		AssistantID: assistantId,
		UserMessage: update.Message.Text,
		UpdateID:    update.UpdateID,
	}

	h.orchestrator.Enqueue(task)
}

func (h *AIHandler) SendMessage(c *gin.Context) {
	userId := c.GetHeader("X-User-Id")
	assistantId := c.GetHeader("X-Assistant-Id")
	userMessage := c.PostForm("user_message")

	if userId == "" || assistantId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized request from gateway"})
		return
	}

	response, err := h.LLM.Conversation(c, userId, assistantId, userMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"answer": response})
}
