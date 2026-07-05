package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// Message represents a chat message (moved from database models)
type Message struct {
	ChatID  uuid.UUID `json:"chat_id"`
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

func (c *Client) GetMessages(assistantId, customerId string) ([]openai.ChatCompletionMessage, error) {
	// 1. Get chat
	chatResp, err := c.db.GetLatestChatByCustomer(assistantId, customerId)
	if err != nil {
		return nil, err
	}

	var messages []openai.ChatCompletionMessage

	// If chat exists, fetch messages
	if chatResp.Id != "" {
		messagesResp, err := c.db.GetAllChatMessages(chatResp.Id)
		if err != nil {
			return nil, err
		}

		var history []Message
		for _, msg := range messagesResp {
			history = append(history, Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		convertedMessages, err := c.ConvertToOpenaiMessage(history)
		if err != nil {
			return nil, err
		}
		messages = append(messages, convertedMessages...)
	}

	// Append system prompt at the end
	startMessages, err := c.StartMessages(assistantId)
	if err != nil {
		return nil, err
	}
	messages = append(messages, startMessages...)

	return messages, nil
}

func (c *Client) StartMessages(assistantId string) ([]openai.ChatCompletionMessage, error) {
	log.Printf("Getting system prompt")

	systemPrompt := "You are a helpful assistant."

	// Try to get dynamic prompt from assistant config via gRPC
	assistant, err := c.db.GetAssistant(assistantId)
	if err != nil {
		return nil, err
	}

	if assistant.Configuration != "" {
		systemPrompt = assistant.Configuration
	}

	// Inject current date/time so the model can resolve relative dates ("this Monday", "next Friday", etc.)
	loc, err := time.LoadLocation("America/Winnipeg")
	now := time.Now()
	if err == nil {
		now = now.In(loc)
	}
	dateContext := fmt.Sprintf("\n\nToday is %s, %s, %s",
		now.Format("Monday"), now.Format("2006-01-02"), now.Format("15:04 MST"))
	systemPrompt += dateContext

	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}, nil
}

func (c *Client) AddMessage(messages *[]openai.ChatCompletionMessage, role string, message string) {
	*messages = append(*messages, openai.ChatCompletionMessage{Role: role, Content: message})
}

func (c *Client) SaveMessages(assistantId, customerId string, messages []openai.ChatCompletionMessage) error {
	log.Printf("Saving messages for user %s", customerId)

	chatResp, err := c.db.GetLatestChatByCustomer(assistantId, customerId)
	if err != nil {
		return err
	}

	chatID := chatResp.Id
	messageCount := chatResp.MessageCount

	if chatID == "" {
		newChat, err := c.db.CreateChat(assistantId, customerId, "openai")
		if err != nil {
			return err
		}
		chatID = newChat.Id
		messageCount = 0
	}

	// Определяем, сколько сообщений уже было (системные + те, что в базе)
	// Нам нужно знать, сколько системных сообщений добавляет StartMessages.
	// Обычно это 1 сообщение.
	systemMessages, _ := c.StartMessages(assistantId)
	systemCount := len(systemMessages)
	
	totalExistingCount := systemCount + int(messageCount)

	// Если новых сообщений в принципе нет, выходим
	if len(messages) <= totalExistingCount {
		return nil
	}

	// Берем только те, которые реально новые
	newCandidateMessages := messages[totalExistingCount:]
	
	for _, msg := range newCandidateMessages {
		// Фильтруем системные сообщения
		if msg.Role == openai.ChatMessageRoleSystem {
			continue
		}

		content := msg.Content
		if content == "" {
			content = " "
		}

		// Если это вызов функции или ответ функции, сохраняем расширенные данные в JSON
		if msg.Role == openai.ChatMessageRoleTool || (msg.Role == openai.ChatMessageRoleAssistant && len(msg.ToolCalls) > 0) {
			type extendedMsg struct {
				Content    string             `json:"content"`
				ToolCallID string             `json:"tool_call_id,omitempty"`
				ToolCalls  []openai.ToolCall  `json:"tool_calls,omitempty"`
			}
			b, err := json.Marshal(extendedMsg{
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				ToolCalls:  msg.ToolCalls,
			})
			if err == nil {
				content = string(b)
			}
		}

		_, err := c.db.SaveMessage(chatID, msg.Role, content, "openai")
		if err != nil {
			log.Printf("Failed to save message: %v", err)
			return fmt.Errorf("failed to save message: %w", err)
		}
	}

	return nil
}

func (c *Client) ConvertToMessage(customerId string, messages []openai.ChatCompletionMessage) []Message {
	var messagesArray []Message

	for _, message := range messages {
		messagesArray = append(messagesArray, Message{
			Role:    message.Role,
			Content: message.Content,
			Time:    time.Now(),
		})
	}

	return messagesArray
}

func (c *Client) ConvertToOpenaiMessage(arrayMessages []Message) ([]openai.ChatCompletionMessage, error) {
	var messages []openai.ChatCompletionMessage

	for _, message := range arrayMessages {
		content := message.Content

		msg := openai.ChatCompletionMessage{
			Role:    message.Role,
			Content: content,
		}

		// Декодируем ToolCallID и ToolCalls, если это tool/assistant сообщение
		if message.Role == openai.ChatMessageRoleTool || message.Role == openai.ChatMessageRoleAssistant {
			type extendedMsg struct {
				Content    string             `json:"content"`
				ToolCallID string             `json:"tool_call_id,omitempty"`
				ToolCalls  []openai.ToolCall  `json:"tool_calls,omitempty"`
			}
			var em extendedMsg
			if err := json.Unmarshal([]byte(content), &em); err == nil {
				if message.Role == openai.ChatMessageRoleTool && em.ToolCallID != "" {
					msg.ToolCallID = em.ToolCallID
					msg.Content = em.Content
				} else if message.Role == openai.ChatMessageRoleAssistant && len(em.ToolCalls) > 0 {
					msg.ToolCalls = em.ToolCalls
					msg.Content = em.Content
				}
			}
		}

		if msg.Content == "" {
			msg.Content = " "
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
