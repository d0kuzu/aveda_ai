package llm

import (
	"context"
	"errors"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func (c *Client) Conversation(ctx context.Context, customerID, assistantID, text string) (string, error) {
	if customerID == "" || assistantID == "" || strings.TrimSpace(text) == "" {
		return "", errors.New("customer, assistant and message are required")
	}
	assistant, err := c.db.GetAssistant(assistantID)
	if err != nil {
		return "", err
	}
	prompt := assistant.GetConfiguration()
	if strings.TrimSpace(prompt) == "" {
		prompt = "You are a helpful assistant."
	}
	messages := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: prompt}}

	chat, err := c.db.GetLatestChatByCustomer(assistantID, customerID)
	if err != nil {
		return "", err
	}
	chatID := chat.GetId()
	if chatID != "" {
		history, err := c.db.GetAllChatMessages(chatID)
		if err != nil {
			return "", err
		}
		for _, entry := range history {
			if entry.Role == openai.ChatMessageRoleUser || entry.Role == openai.ChatMessageRoleAssistant {
				messages = append(messages, openai.ChatCompletionMessage{Role: entry.Role, Content: entry.Content})
			}
		}
	}
	messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: text})
	response, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
	})
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
		return "", errors.New("model returned an empty response")
	}
	answer := response.Choices[0].Message.Content
	platform := assistant.GetType()
	if chatID == "" {
		chat, err = c.db.CreateChat(assistantID, customerID, platform)
		if err != nil {
			return "", err
		}
		chatID = chat.GetId()
	}
	if _, err := c.db.SaveMessage(chatID, openai.ChatMessageRoleUser, text, platform); err != nil {
		return "", err
	}
	if _, err := c.db.SaveMessage(chatID, openai.ChatMessageRoleAssistant, answer, platform); err != nil {
		return "", err
	}
	return answer, nil
}
