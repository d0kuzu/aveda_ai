package ws

import (
	"encoding/json"
	"log"
	"strings"
	"time"
)

func (c *Client) PollLocalDB(chatID string) {
	var lastMessageCount int

	for {
		select {
		case <-c.done:
			return
		default:
		}

		messages, err := c.Db.GetChatMessages(chatID, 50, int32(lastMessageCount))
		if err != nil {
			log.Println("DB fetch error:", err)
			select {
			case <-c.done:
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

				body := msg.Content

				// Если это JSON ответа с вызовом функции (assistant с tool_calls)
				if author == "bot" && strings.HasPrefix(body, "{") && strings.Contains(body, "\"tool_calls\"") {
					var extMsg struct {
						Content string `json:"content"`
					}
					if err := json.Unmarshal([]byte(body), &extMsg); err == nil {
						body = extMsg.Content
					}
				}

				body = strings.TrimSpace(body)
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

				c.Broadcast(chatID, data)
			}
			lastMessageCount += newCount
		}

		select {
		case <-c.done:
			return
		case <-time.After(1 * time.Second):
		}
	}
}
