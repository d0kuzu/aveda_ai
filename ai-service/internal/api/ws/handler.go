package ws

import (
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/ws"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	cfg *config.Settings
	db  *db.Client
}

func NewWSHandler(cfg *config.Settings, db *db.Client) *WSHandler {
	return &WSHandler{cfg: cfg, db: db}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *WSHandler) ChatPolling(c *gin.Context) {
	chatID := c.Query("chat")
	if chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat id required"})
		return
	}

	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-Id header is required"})
		return
	}

	// Fetch chat to get assistant_id
	chatResp, err := h.db.GetChat(chatID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat not found"})
		return
	}

	// Fetch user's assistants
	assistants, err := h.db.GetAssistantsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch assistants"})
		return
	}

	// Validate ownership
	isOwner := false
	for _, a := range assistants {
		if a.Id == chatResp.AssistantId {
			isOwner = true
			break
		}
	}

	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied: chat does not belong to user"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	client := ws.NewWSClient(conn, chatID, h.db)
	ws.RegisterClient(client)
	log.Printf("new client connected to chat %s", chatID)

	go client.PollLocalDB(chatID)

	go client.Listen()
}
