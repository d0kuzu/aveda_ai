package chat

import (
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	dbpb "diaxel/proto/db"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	cfg *config.Settings
	db  *db.Client
}

type ChatJSON struct {
	ID            string `json:"id"`
	AssistantID   string `json:"assistant_id"`
	CustomerID    string `json:"customer_id"`
	Platform      string `json:"platform"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	MessageCount  int32  `json:"message_count"`
	IsEnd         bool   `json:"is_end"`
	FollowupStage int32  `json:"followup_stage"`
	IsReviewed    bool   `json:"is_reviewed"`
	IsBooked      bool   `json:"is_booked"`
}

func chatToJSON(chat *dbpb.ChatResponse) ChatJSON {
	return ChatJSON{
		ID:            chat.Id,
		AssistantID:   chat.AssistantId,
		CustomerID:    chat.CustomerId,
		Platform:      chat.Platform,
		CreatedAt:     chat.CreatedAt,
		UpdatedAt:     chat.UpdatedAt,
		MessageCount:  chat.MessageCount,
		IsEnd:         chat.IsEnd,
		FollowupStage: chat.FollowupStage,
		IsReviewed:    chat.IsReviewed,
		IsBooked:      chat.IsBooked,
	}
}

func NewChatHandler(cfg *config.Settings, db *db.Client) *ChatHandler {
	return &ChatHandler{cfg: cfg, db: db}
}

func (h *ChatHandler) getValidatedAssistantIDs(c *gin.Context, userID string) ([]string, error) {
	assistants, err := h.db.GetAssistantsByUserID(userID)
	if err != nil {
		log.Printf("[getValidatedAssistantIDs] Error getting assistants for user %s: %v", userID, err)
		return nil, err
	}
	log.Printf("[getValidatedAssistantIDs] Found %d assistants for user %s", len(assistants), userID)

	validAssistantIDsMap := make(map[string]bool)
	var userAssistantIDs []string
	for _, a := range assistants {
		validAssistantIDsMap[a.Id] = true
		userAssistantIDs = append(userAssistantIDs, a.Id)
	}

	var finalAssistantIDs []string
	if assistantsParam := c.Query("assistant_ids"); assistantsParam != "" {
		requestedIDs := strings.Split(assistantsParam, ",")
		for _, id := range requestedIDs {
			if !validAssistantIDsMap[id] {
				return nil, fmt.Errorf("указанный assistant_id (%s) не принадлежит пользователю", id)
			}
			finalAssistantIDs = append(finalAssistantIDs, id)
		}
	} else {
		finalAssistantIDs = userAssistantIDs
	}

	log.Printf("[getValidatedAssistantIDs] Final assistant IDs: %v", finalAssistantIDs)
	return finalAssistantIDs, nil
}

func (h *ChatHandler) GetAllChats(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-Id header is required"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.ParseInt(pageStr, 10, 32)
	if err != nil || page < 1 {
		page = 1
	}

	assistantIDs, err := h.getValidatedAssistantIDs(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate assistants", "details": err.Error()})
		return
	}

	if len(assistantIDs) == 0 {
		log.Printf("[GetAllChats] No assistant IDs for user %s, returning empty result", userID)
		c.JSON(http.StatusOK, gin.H{"answer": []ChatJSON{}})
		return
	}

	log.Printf("[GetAllChats] Fetching chats for assistants: %v, page: %d", assistantIDs, page)

	chatsPerPage := int32(10)

	chats, err := h.db.GetChatPageByUserID(assistantIDs, int32(page), chatsPerPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch chats", "details": err.Error()})
		return
	}

	result := make([]ChatJSON, len(chats))
	for i, chat := range chats {
		result[i] = chatToJSON(chat)
	}

	c.JSON(http.StatusOK, gin.H{
		"answer": result,
	})
}

func (h *ChatHandler) GetPagination(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-Id header is required"})
		return
	}

	assistantIDs, err := h.getValidatedAssistantIDs(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate assistants", "details": err.Error()})
		return
	}

	if len(assistantIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"answer": 0})
		return
	}

	chatsPerPage := int32(10)
	pagesCount, err := h.db.GetChatPagesCountByUserID(assistantIDs, chatsPerPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pagination", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"answer": pagesCount})
}

func (h *ChatHandler) GetChat(c *gin.Context) {
	chatID := c.Query("chat")
	if chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat parameter is required (chatID)"})
		return
	}

	messages, err := h.db.GetAllChatMessages(chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch chat messages", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"answer": gin.H{
		"chat_id":  chatID,
		"messages": messages,
		"count":    len(messages),
	}})
}

func (h *ChatHandler) SearchChat(c *gin.Context) {
	searchTerm := c.Query("chat")

	if searchTerm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat parameter is required (search term)"})
		return
	}

	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-Id header is required"})
		return
	}

	assistantIDs, err := h.getValidatedAssistantIDs(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate assistants", "details": err.Error()})
		return
	}

	if len(assistantIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"answer": []string{}, "total_count": 0})
		return
	}

	chats, totalCount, err := h.db.SearchChatsByCustomer(assistantIDs, searchTerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search chats", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"answer":      chats,
		"total_count": totalCount,
	})
}

// ClearAll is a test endpoint to remove all chats and messages
func (h *ChatHandler) ClearAll(c *gin.Context) {
	if err := h.db.DeleteAllChatsAndMessages(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear all chats and messages", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "all chats and messages have been cleared"})
}
