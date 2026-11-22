package ws

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func (c *Client) PollLocalDB(chatID string) {
	var lastMessageCount int

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		messages, err := c.Db.GetChatMessages(c.ctx, chatID, 50, int32(lastMessageCount))
		if err != nil {
			log.Println("DB fetch error:", err)
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		newCount := len(messages)

		if newCount > 0 {
			for _, msg := range messages {
				var author string
				switch msg.Role {
				case "user":
					author = "client"
				case "assistant":
					author = "bot"
				default:
					continue
				}

				body := strings.TrimSpace(msg.Content)
				if body == "" {
					continue
				}

				wsMsg := Message{
					Author: author,
					Body:   body,
				}

				data, err := json.Marshal(wsMsg)
				if err != nil {
					log.Println("ws message json marshal error:", err)
					continue
				}

				if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Println("ws message write error:", err)
					c.cancel()
					c.conn.Close()
					return
				}
			}
			lastMessageCount += newCount
		}

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	}
}
