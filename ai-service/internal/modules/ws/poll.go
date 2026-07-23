package ws

import (
	"encoding/json"
	"log"
	"time"
)

func (c *Client) PollLocalDB(chatID string) {
	var lastMessageCount int

	for {
		messages, err := c.Db.GetChatMessages(chatID, 50, int32(lastMessageCount))
		if err != nil {
			log.Println("DB fetch error:", err)
			time.Sleep(5 * time.Second)
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

				wsMsg := Message{
					Author: author,
					Body:   msg.Content,
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

		time.Sleep(500 * time.Millisecond)
	}
}
