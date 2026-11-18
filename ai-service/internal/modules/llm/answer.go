package llm

import (
	"context"
	"diaxel/internal/constants"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func (c *Client) GetAnswer(ctx context.Context, assistantId string, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	isTutorMode := false
	for _, msg := range messages {
		if msg.Role == openai.ChatMessageRoleSystem && strings.Contains(msg.Content, "State Board Exam Prep Tutor") {
			isTutorMode = true
			break
		}
	}

	var tools []openai.Tool
	if isTutorMode || assistantId == constants.AvedaTutorAssistantID {
		tools = constants.TutorTools
	} else {
		switch assistantId {
		case constants.AvedaSintaAssistantID:
			tools = constants.AvedaSintaTools
		case constants.AvedaCanadaAssistantID:
			tools = constants.AvedaCanadaTools
		default:
			tools = nil
		}
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
