package llm

import (
	"fmt"
	"log"
	"strings"
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
	now := time.Now()
	dateContext := fmt.Sprintf("\n\nToday is %s, %s",
		now.Format("Monday"), now.Format("2006-01-02"))
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
		// Фильтруем системные и технические сообщения (на всякий случай)
		if msg.Role == openai.ChatMessageRoleSystem || 
		   msg.Role == openai.ChatMessageRoleTool ||
		   (msg.Role == openai.ChatMessageRoleAssistant && (msg.Content == "" || msg.Content == " ") && len(msg.ToolCalls) > 0) {
			continue
		}

		content := msg.Content
		if content == "" {
			content = " "
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
		if content == "" {
			content = " "
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    message.Role,
			Content: content,
		})
	}

	return messages, nil
}

func (c *Client) RemoveSystemMessages(messages *[]openai.ChatCompletionMessage) {
	var otherMessages []openai.ChatCompletionMessage

	for _, message := range *messages {
		if message.Role != "system" || strings.Contains(message.Content, "#function_call") {
			otherMessages = append(otherMessages, message)
		}
	}

	*messages = otherMessages
}
