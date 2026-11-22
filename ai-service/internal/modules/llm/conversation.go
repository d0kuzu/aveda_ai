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
	assistant, err := c.db.GetAssistant(ctx, assistantID)
	if err != nil {
		return "", err
	}
	prompt := assistant.Configuration
	if strings.TrimSpace(prompt) == "" {
		prompt = "You are a helpful assistant."
	}
	messages := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: prompt}}

	chat, err := c.db.GetLatestChatByCustomer(ctx, assistantID, customerID)
	if err != nil {
		return "", err
	}
	if chat != nil {
		history, err := c.db.GetAllChatMessages(ctx, chat.Id)
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
	if err := c.db.SaveConversationTurn(ctx, assistantID, customerID, text, answer); err != nil {
		return "", err
	}
	return answer, nil
}
