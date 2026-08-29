package llm

import (
	"context"
	"diaxel/internal/constants"

	"github.com/sashabaranov/go-openai"
)

func (c *Client) GetAnswer(ctx context.Context, assistantId string, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	var tools []openai.Tool
	switch assistantId {
	case constants.AvedaSintaAssistantID:
		tools = constants.AvedaSintaTools
	case constants.AvedaCanadaAssistantID:
		tools = constants.AvedaCanadaTools
	default:
		tools = nil
	}

	response, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
	})
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	return response, nil
}
