package llm

import (
	"context"
	"diaxel/internal/constants"

	"github.com/sashabaranov/go-openai"
)

func (c *Client) GetAnswer(ctx context.Context, assistantId string, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	var tools []openai.Tool
	switch assistantId {
	case "7de674c1-6029-4609-92c4-074fab740bad":
		tools = constants.AvedaSintaTools
	case "194f277c-0911-4ca7-bba2-665036627bc0":
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
