package ws

import (
	"context"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/twilio"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	chat string
	Db   *db.Client
	done chan struct{}
}

func NewWSClient(conn *websocket.Conn, chatID string, db *db.Client) *Client {
	return &Client{conn: conn, chat: chatID, Db: db, done: make(chan struct{})}
}

var (
	clients   = make(map[*Client]bool)
	clientsMu sync.Mutex
)

func RegisterClient(c *Client) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	clients[c] = true
}

func UnregisterClient(c *Client) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	delete(clients, c)
	c.conn.Close()
	select {
	case <-c.done:
	default:
		close(c.done)
	}
}

func (c *Client) Broadcast(chatID string, msg []byte) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	err := c.conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Println("ws message write error:", err)
		c.conn.Close()
		delete(clients, c)
		select {
		case <-c.done:
		default:
			close(c.done)
		}
	}
}

func (c *Client) Listen() {
	defer UnregisterClient(c)

	twClient := twilio.InitClient()

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("client disconnected:", err)
			return
		}

		// Save the message to DB
		_, err = c.Db.SaveMessage(c.chat, "assistant", string(msg), "twilio")
		if err != nil {
			log.Println("ws listen: failed to save message:", err)
		}

		chat, err := c.Db.GetChat(c.chat)
		if err != nil || chat == nil {
			log.Println("ws listen: failed to get chat:", err)
			continue
		}

		twConfig, err := c.Db.GetTwilioConfig(chat.AssistantId)
		if err != nil || twConfig == nil {
			log.Println("ws listen: failed to get twilio config:", err)
			continue
		}

		err = twClient.SendMessage(context.Background(), twConfig.AccountSid, twConfig.AuthToken, twConfig.TwilioNumber, chat.CustomerId, string(msg))
		if err != nil {
			log.Println("ws listen: failed to send twilio message:", err)
		}
	}
}
