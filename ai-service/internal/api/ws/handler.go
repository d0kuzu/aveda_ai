package ws

import (
	"log"
	"net/http"

	"diaxel/internal/api/access"
	"diaxel/internal/modules/database"
	"diaxel/internal/modules/twilio"
	"diaxel/internal/modules/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	db     *database.Store
	twilio *twilio.Client
}

func NewWSHandler(db *database.Store, twilioClient *twilio.Client) *WSHandler {
	return &WSHandler{db: db, twilio: twilioClient}
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

	if !access.Chat(c, h.db, chatID) {
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	client := ws.NewWSClient(c.Request.Context(), conn, chatID, h.db, h.twilio)
	log.Printf("new client connected to chat %s", chatID)

	go client.PollLocalDB(chatID)
	client.Listen()
}
