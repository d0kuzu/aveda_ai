package ws

import (
	"context"
	"log"

	"diaxel/internal/modules/database"
	"diaxel/internal/modules/twilio"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	chat   string
	Db     *database.Store
	twilio *twilio.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func NewWSClient(ctx context.Context, conn *websocket.Conn, chatID string, db *database.Store, twilioClient *twilio.Client) *Client {
	ctx, cancel := context.WithCancel(ctx)
	return &Client{conn: conn, chat: chatID, Db: db, twilio: twilioClient, ctx: ctx, cancel: cancel}
}

func (c *Client) Listen() {
	defer c.conn.Close()
	defer c.cancel()

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("client disconnected:", err)
			return
		}

		chat, err := c.Db.GetChat(c.ctx, c.chat)
		if err != nil || chat == nil {
			log.Println("ws listen: failed to get chat:", err)
			continue
		}

		assistant, err := c.Db.GetAssistant(c.ctx, chat.AssistantId)
		if err != nil {
			log.Println("ws listen: failed to get assistant:", err)
			continue
		}
		if assistant.Type == "twilio" {
			config, err := c.Db.GetTwilioConfig(c.ctx, chat.AssistantId)
			if err != nil || config == nil {
				log.Println("ws listen: failed to get Twilio config:", err)
				continue
			}
			if err := c.twilio.SendMessage(c.ctx, config.AccountSid, config.AuthToken, config.TwilioNumber, chat.CustomerId, string(msg)); err != nil {
				log.Println("ws listen: failed to send SMS:", err)
				continue
			}
		}
		if _, err := c.Db.SaveMessage(c.ctx, c.chat, "assistant", string(msg), assistant.Type); err != nil {
			log.Println("ws listen: failed to save message:", err)
		}
	}
}
